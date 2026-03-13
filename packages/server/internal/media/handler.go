package media

import (
	"context"
	"errors"
	"io"
	"log"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type AssetURLResponse struct {
	AssetID   uuid.UUID `json:"assetId"`
	URL       string    `json:"url"`
	ExpiresAt string    `json:"expiresAt"`
}

type UploadAssetResponse struct {
	ID              uuid.UUID `json:"id"`
	AssetID         uuid.UUID `json:"assetId"`
	DocumentID      uuid.UUID `json:"documentId"`
	Kind            string    `json:"kind"`
	Filename        string    `json:"filename"`
	MimeType        string    `json:"mimeType"`
	FileSize        int64     `json:"fileSize"`
	StorageProvider string    `json:"storageProvider"`
	ObjectKey       string    `json:"objectKey"`
	URL             string    `json:"url"`
	ExpiresAt       string    `json:"expiresAt,omitempty"`
	Visibility      string    `json:"visibility"`
}

type AssetReferencesResponse struct {
	AssetID        uuid.UUID                `json:"assetId"`
	ReferenceCount int                      `json:"referenceCount"`
	Documents      []AssetReferenceDocument `json:"documents"`
}

type AssetListResponse struct {
	Items   []AssetListItem `json:"items"`
	HasMore bool            `json:"hasMore"`
	Total   int64           `json:"total"`
}

func GetAssetURLHandler(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "Unauthorized",
			Message: "Invalid user context",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid user id",
		})
	}

	assetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid asset id",
		})
	}

	if _, err := GetOwnedAsset(userID, assetID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "Not Found",
			Message: err.Error(),
		})
	}

	tokenService, err := NewTokenService()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	token, expiresAt, err := tokenService.IssueAssetReadToken(assetID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Failed to issue media token",
		})
	}

	readURL := c.BaseURL() + "/api/v1/media/assets/" + assetID.String() + "/content?token=" + url.QueryEscape(token)
	return c.JSON(AssetURLResponse{
		AssetID:   assetID,
		URL:       readURL,
		ExpiresAt: expiresAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

func ListAssetsHandler(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "Unauthorized",
			Message: "Invalid user context",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid user id",
		})
	}

	result, err := ListOwnedAssets(ListAssetsRequest{
		UserID: userID,
		Kind:   c.Query("kind"),
		Status: c.Query("status"),
		Query:  c.Query("q"),
		Limit:  c.QueryInt("limit", 20),
		Offset: c.QueryInt("offset", 0),
	})
	if err != nil {
		switch err.Error() {
		case "invalid asset status", "invalid asset kind":
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:   "Bad Request",
				Message: err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error:   "Internal Server Error",
				Message: err.Error(),
			})
		}
	}

	return c.JSON(AssetListResponse{
		Items:   result.Items,
		HasMore: result.HasMore,
		Total:   result.Total,
	})
}

func GetAssetContentHandler(c *fiber.Ctx) error {
	assetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid asset id",
		})
	}

	asset, err := GetAssetByID(assetID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "Not Found",
			Message: err.Error(),
		})
	}

	if asset.Visibility != "public" {
		token := c.Query("token")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error:   "Unauthorized",
				Message: "Missing media token",
			})
		}

		tokenService, err := NewTokenService()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error:   "Internal Server Error",
				Message: err.Error(),
			})
		}

		claims, err := tokenService.VerifyAssetReadToken(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error:   "Unauthorized",
				Message: "Invalid media token: " + err.Error(),
			})
		}

		if claims.AssetID != asset.ID {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error:   "Unauthorized",
				Message: "Media token does not match asset",
			})
		}

		// Do not hard-bind user in the read token check.
		// Token is already short-lived and signed server-side; asset binding is enough here.
	}

	if err := initStorageProvider(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	obj, err := storageProvider.GetObject(context.Background(), asset.ObjectKey)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(ErrorResponse{
			Error:   "Media Upstream Error",
			Message: err.Error(),
		})
	}
	defer obj.Body.Close()

	contentType := asset.MimeType
	if contentType == "" {
		contentType = obj.ContentType
	}
	c.Set("Content-Type", contentType)
	c.Set("Cache-Control", "private, max-age=60")

	data, err := io.ReadAll(obj.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Failed to read asset content",
		})
	}
	return c.Send(data)
}

func GetAssetReferencesHandler(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "Unauthorized",
			Message: "Invalid user context",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid user id",
		})
	}

	assetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid asset id",
		})
	}

	result, err := GetOwnedAssetReferences(userID, assetID)
	if err != nil {
		if err.Error() == "资源不存在或无权访问" {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error:   "Not Found",
				Message: err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	return c.JSON(AssetReferencesResponse{
		AssetID:        result.AssetID,
		ReferenceCount: result.ReferenceCount,
		Documents:      result.Documents,
	})
}

func DeleteAssetHandler(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "Unauthorized",
			Message: "Invalid user context",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid user id",
		})
	}

	assetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid asset id",
		})
	}

	if err := DeleteOwnedUnusedAsset(context.Background(), userID, assetID); err != nil {
		switch err.Error() {
		case "资源不存在或无权访问":
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error:   "Not Found",
				Message: err.Error(),
			})
		case "asset is still referenced by documents", "asset already deleted":
			return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
				Error:   "Conflict",
				Message: err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error:   "Internal Server Error",
				Message: err.Error(),
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func ValidateVisibility(visibility string) error {
	switch visibility {
	case "", "private", "public":
		return nil
	default:
		return errors.New("invalid visibility")
	}
}

// UploadDocumentAssetHandler handles POST /api/v1/edit/documents/:id/assets
func UploadDocumentAssetHandler(c *fiber.Ctx) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[media.upload] panic: %v", r)
			_ = c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error:   "Upload Failed",
				Message: "upload handler panic",
			})
		}
	}()

	userIDStr, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "Unauthorized",
			Message: "Invalid user context",
		})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid user id",
		})
	}
	documentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid document id",
		})
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "file is required",
		})
	}

	visibility := c.FormValue("visibility")
	log.Printf("[media.upload] start document=%s user=%s filename=%q size=%d visibility=%q", documentID, userID, fileHeader.Filename, fileHeader.Size, visibility)
	result, err := UploadDocumentAsset(context.Background(), UploadAssetRequest{
		DocumentID: documentID,
		UserID:     userID,
		FileHeader: fileHeader,
		Visibility: visibility,
	})
	if err != nil {
		log.Printf("[media.upload] failed document=%s user=%s filename=%q err=%v", documentID, userID, fileHeader.Filename, err)
		status := fiber.StatusInternalServerError
		switch err.Error() {
		case "文档不存在或无权访问", "file is required", "invalid visibility":
			status = fiber.StatusBadRequest
		default:
			if errors.Is(err, context.Canceled) {
				status = fiber.StatusRequestTimeout
			}
			if len(err.Error()) >= len("unsupported file type:") && err.Error()[:len("unsupported file type:")] == "unsupported file type:" {
				status = fiber.StatusBadRequest
			}
		}
		return c.Status(status).JSON(ErrorResponse{
			Error:   "Upload Failed",
			Message: err.Error(),
		})
	}

	asset := result.Asset
	docID := documentID
	if asset.DocumentID != nil {
		docID = *asset.DocumentID
	}

	readURL := c.BaseURL() + "/api/v1/media/assets/" + asset.ID.String() + "/content"
	expiresAt := ""
	if asset.Visibility != "public" {
		tokenService, err := NewTokenService()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error:   "Internal Server Error",
				Message: err.Error(),
			})
		}
		token, exp, err := tokenService.IssueAssetReadToken(asset.ID, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error:   "Internal Server Error",
				Message: "Failed to issue media token",
			})
		}
		readURL = readURL + "?token=" + url.QueryEscape(token)
		expiresAt = exp.UTC().Format("2006-01-02T15:04:05Z")
	}

	log.Printf("[media.upload] success asset=%s provider=%s objectKey=%q", asset.ID, asset.StorageProvider, asset.ObjectKey)
	return c.Status(fiber.StatusCreated).JSON(UploadAssetResponse{
		ID:              asset.ID,
		AssetID:         asset.ID,
		DocumentID:      docID,
		Kind:            asset.Kind,
		Filename:        asset.Filename,
		MimeType:        asset.MimeType,
		FileSize:        asset.FileSize,
		StorageProvider: asset.StorageProvider,
		ObjectKey:       asset.ObjectKey,
		URL:             readURL,
		ExpiresAt:       expiresAt,
		Visibility:      asset.Visibility,
	})
}
