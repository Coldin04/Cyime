package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	documentImageModeManagedAsset = "managed_asset"
	documentImageModeExternalURL  = "external_url"

	documentImageTargetManagedR2 = "managed-r2"
)

var seeHTTPClient = &http.Client{}
var lskyHTTPClient = &http.Client{}

type UploadDocumentImageRequest struct {
	DocumentID uuid.UUID
	UserID     uuid.UUID
	FileHeader *multipart.FileHeader
}

type UploadDocumentImageResult struct {
	TargetID string     `json:"targetId"`
	Mode     string     `json:"mode"`
	URL      string     `json:"url"`
	AssetID  *uuid.UUID `json:"assetId,omitempty"`
}

type documentImageUploader interface {
	Upload(ctx context.Context, req UploadDocumentImageRequest) (*UploadDocumentImageResult, error)
}

type managedDocumentImageUploader struct{}

func (u *managedDocumentImageUploader) Upload(ctx context.Context, req UploadDocumentImageRequest) (*UploadDocumentImageResult, error) {
	result, err := UploadDocumentAsset(ctx, UploadAssetRequest{
		DocumentID: req.DocumentID,
		UserID:     req.UserID,
		FileHeader: req.FileHeader,
		Visibility: "private",
	})
	if err != nil {
		return nil, err
	}

	return &UploadDocumentImageResult{
		TargetID: documentImageTargetManagedR2,
		Mode:     documentImageModeManagedAsset,
		URL:      result.Asset.URL,
		AssetID:  &result.Asset.ID,
	}, nil
}

type seeDocumentImageUploader struct {
	targetID   string
	apiBaseURL string
	apiToken   string
}

type seeUploadResponse struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Images  string `json:"images"`
	Data    struct {
		URL string `json:"url"`
	} `json:"data"`
}

func (u *seeDocumentImageUploader) Upload(ctx context.Context, req UploadDocumentImageRequest) (*UploadDocumentImageResult, error) {
	if strings.TrimSpace(u.apiToken) == "" {
		return nil, errors.New("S.EE API token is not configured")
	}
	if req.FileHeader == nil {
		return nil, errors.New("file is required")
	}

	file, err := req.FileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("smfile", req.FileHeader.Filename)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(u.apiBaseURL, "/")+"/upload",
		body,
	)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", u.apiToken)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := seeHTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload seeUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		if payload.Message != "" {
			return nil, fmt.Errorf("S.EE upload failed: %s", payload.Message)
		}
		return nil, fmt.Errorf("S.EE upload failed with status %d", resp.StatusCode)
	}

	uploadedURL := strings.TrimSpace(payload.Data.URL)
	if uploadedURL == "" {
		uploadedURL = strings.TrimSpace(payload.Images)
	}
	if uploadedURL == "" {
		if payload.Message != "" {
			return nil, fmt.Errorf("S.EE upload failed: %s", payload.Message)
		}
		return nil, errors.New("S.EE upload did not return a usable URL")
	}

	return &UploadDocumentImageResult{
		TargetID: u.targetID,
		Mode:     documentImageModeExternalURL,
		URL:      uploadedURL,
	}, nil
}

type lskyDocumentImageUploader struct {
	targetID  string
	apiURL    string
	apiToken  string
	storageID int
	strategy  string
}

type lskyUploadResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		PublicURL string `json:"public_url"`
		URL       string `json:"url"`
	} `json:"data"`
}

func (u *lskyDocumentImageUploader) Upload(ctx context.Context, req UploadDocumentImageRequest) (*UploadDocumentImageResult, error) {
	if strings.TrimSpace(u.apiToken) == "" {
		return nil, errors.New("Lsky API token is not configured")
	}
	if strings.TrimSpace(u.apiURL) == "" {
		return nil, errors.New("Lsky API url is not configured")
	}
	if u.storageID <= 0 {
		return nil, errors.New("Lsky storage id is not configured")
	}
	if req.FileHeader == nil {
		return nil, errors.New("file is required")
	}

	file, err := req.FileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", req.FileHeader.Filename)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	if err := writer.WriteField("storage_id", fmt.Sprintf("%d", u.storageID)); err != nil {
		return nil, err
	}
	if strings.TrimSpace(u.strategy) != "" {
		if err := writer.WriteField("strategy_id", strings.TrimSpace(u.strategy)); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.apiURL, body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+u.apiToken)
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := lskyHTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload lskyUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 || !strings.EqualFold(strings.TrimSpace(payload.Status), "success") {
		if payload.Message != "" {
			return nil, fmt.Errorf("Lsky upload failed: %s", payload.Message)
		}
		return nil, errors.New("Lsky upload failed")
	}

	uploadedURL := strings.TrimSpace(payload.Data.PublicURL)
	if uploadedURL == "" {
		uploadedURL = strings.TrimSpace(payload.Data.URL)
	}
	if uploadedURL == "" {
		return nil, errors.New("Lsky upload did not return a usable URL")
	}

	return &UploadDocumentImageResult{
		TargetID: u.targetID,
		Mode:     documentImageModeExternalURL,
		URL:      uploadedURL,
	}, nil
}

func getDocumentImageUploadTargetID(userID, documentID uuid.UUID) (string, error) {
	var document models.Document
	result := database.DB.
		Select("id", "preferred_image_target_id").
		Where("id = ? AND owner_user_id = ? AND deleted_at IS NULL", documentID, userID).
		First(&document)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return "", errors.New("文档不存在或无权访问")
		}
		return "", result.Error
	}

	switch strings.TrimSpace(document.PreferredImageTargetID) {
	case "":
		return documentImageTargetManagedR2, nil
	case documentImageTargetManagedR2:
		return document.PreferredImageTargetID, nil
	default:
		if _, err := uuid.Parse(strings.TrimSpace(document.PreferredImageTargetID)); err != nil {
			return "", errors.New("document image target is not supported")
		}
		return strings.TrimSpace(document.PreferredImageTargetID), nil
	}
}

func newDocumentImageUploader(userID uuid.UUID, targetID string) (documentImageUploader, error) {
	switch targetID {
	case documentImageTargetManagedR2:
		return &managedDocumentImageUploader{}, nil
	default:
		configID, err := uuid.Parse(targetID)
		if err != nil {
			return nil, errors.New("document image target is not supported")
		}
		config, err := getUserImageBedConfig(userID, configID)
		if err != nil {
			return nil, err
		}
		switch config.ProviderType {
		case "see":
			apiToken, err := getEffectiveSeeAPIToken(config)
			if err != nil {
				return nil, err
			}
			return &seeDocumentImageUploader{
				targetID:   config.ID.String(),
				apiBaseURL: getSeeAPIBaseURL(),
				apiToken:   apiToken,
			}, nil
		case "lsky":
			apiURL, apiToken, storageID, strategy, err := buildLskyUploadConfig(config)
			if err != nil {
				return nil, err
			}
			return &lskyDocumentImageUploader{
				targetID:  config.ID.String(),
				apiURL:    apiURL,
				apiToken:  apiToken,
				storageID: storageID,
				strategy:  strategy,
			}, nil
		default:
			return nil, errors.New("document image target is not supported")
		}
	}
}

func getSeeAPIBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("SEE_API_BASE_URL")); value != "" {
		return value
	}
	return "https://s.ee/api/v1/file"
}

func getUserImageBedConfig(userID, configID uuid.UUID) (*models.UserImageBedConfig, error) {
	var config models.UserImageBedConfig
	if err := database.DB.
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", configID, userID).
		First(&config).Error; err != nil {
		return nil, err
	}
	if !config.IsEnabled {
		return nil, errors.New("image bed config is disabled")
	}
	return &config, nil
}

func buildLskyUploadConfig(config *models.UserImageBedConfig) (string, string, int, string, error) {
	apiURL := strings.TrimRight(strings.TrimSpace(stringPtrValue(config.BaseURL)), "/")
	if apiURL == "" {
		apiURL = strings.TrimRight(strings.TrimSpace(os.Getenv("LSKY_API_URL")), "/")
	}
	if apiURL == "" {
		return "", "", 0, "", errors.New("Lsky API url is not configured")
	}
	if strings.HasSuffix(apiURL, "/upload") {
		// keep as-is
	} else if strings.HasSuffix(apiURL, "/api/v1") || strings.HasSuffix(apiURL, "/api/v2") {
		apiURL = apiURL + "/upload"
	} else {
		apiURL = apiURL + "/api/v2/upload"
	}

	apiToken := strings.TrimSpace(stringPtrValue(config.APIToken))
	if apiToken == "" {
		apiToken = strings.TrimSpace(os.Getenv("LSKY_API_TOKEN"))
	}
	if apiToken == "" {
		return "", "", 0, "", errors.New("Lsky API token is not configured")
	}

	storageID := 0
	strategy := ""
	if config.ConfigJSON != nil && strings.TrimSpace(*config.ConfigJSON) != "" {
		var payload struct {
			StorageID  int    `json:"storageId"`
			StrategyID string `json:"strategyId"`
		}
		if err := json.Unmarshal([]byte(*config.ConfigJSON), &payload); err == nil {
			storageID = payload.StorageID
			strategy = strings.TrimSpace(payload.StrategyID)
		}
	}
	if storageID <= 0 {
		return "", "", 0, "", errors.New("Lsky storage id is not configured")
	}

	return apiURL, apiToken, storageID, strategy, nil
}

func getEffectiveSeeAPIToken(config *models.UserImageBedConfig) (string, error) {
	if value := strings.TrimSpace(stringPtrValue(config.APIToken)); value != "" {
		return value, nil
	}
	if value := strings.TrimSpace(os.Getenv("SEE_API_TOKEN")); value != "" {
		return value, nil
	}
	return "", errors.New("S.EE API token is not configured")
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func UploadDocumentImage(ctx context.Context, req UploadDocumentImageRequest) (*UploadDocumentImageResult, error) {
	if req.FileHeader == nil {
		return nil, errors.New("file is required")
	}

	targetID, err := getDocumentImageUploadTargetID(req.UserID, req.DocumentID)
	if err != nil {
		return nil, err
	}

	uploader, err := newDocumentImageUploader(req.UserID, targetID)
	if err != nil {
		return nil, err
	}

	return uploader.Upload(ctx, req)
}
