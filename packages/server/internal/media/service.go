package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UploadAssetRequest struct {
	DocumentID uuid.UUID
	UserID     uuid.UUID
	FileHeader *multipart.FileHeader
	Visibility string
}

type UploadAssetResult struct {
	Asset *models.Asset
}

type AssetReferenceDocument struct {
	DocumentID uuid.UUID `json:"documentId"`
	Title      string    `json:"title"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type AssetReferencesResult struct {
	AssetID        uuid.UUID                `json:"assetId"`
	ReferenceCount int                      `json:"referenceCount"`
	Documents      []AssetReferenceDocument `json:"documents"`
}

type ListAssetsRequest struct {
	UserID uuid.UUID
	Kind   string
	Status string
	Query  string
	Limit  int
}

type AssetListItem struct {
	ID             uuid.UUID  `json:"id"`
	Kind           string     `json:"kind"`
	Filename       string     `json:"filename"`
	MimeType       string     `json:"mimeType"`
	FileSize       int64      `json:"fileSize"`
	Visibility     string     `json:"visibility"`
	Status         string     `json:"status"`
	ReferenceCount int        `json:"referenceCount"`
	Deletable      bool       `json:"deletable"`
	DocumentID     *uuid.UUID `json:"documentId,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type ListAssetsResult struct {
	Items []AssetListItem `json:"items"`
}

var storageProvider StorageProvider

var allowedAssetMimeTypes = map[string]struct{}{
	"image/png":  {},
	"image/jpeg": {},
	"image/webp": {},
	"image/gif":  {},
	"video/mp4":  {},
	"video/webm": {},
}

var allowedAssetExtensions = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".gif":  "image/gif",
	".mp4":  "video/mp4",
	".webm": "video/webm",
}

func initStorageProvider() error {
	if storageProvider != nil {
		return nil
	}
	provider, err := newStorageProviderFromEnv()
	if err != nil {
		return err
	}
	storageProvider = provider
	return nil
}

func GetOwnedAsset(userID, assetID uuid.UUID) (*models.Asset, error) {
	var asset models.Asset
	result := database.DB.Where("id = ? AND owner_user_id = ? AND deleted_at IS NULL", assetID, userID).First(&asset)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("资源不存在或无权访问")
		}
		return nil, result.Error
	}
	return &asset, nil
}

func ListOwnedAssets(req ListAssetsRequest) (*ListAssetsResult, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	query := database.DB.Model(&models.Asset{}).
		Where("owner_user_id = ?", req.UserID)

	status := strings.TrimSpace(req.Status)
	switch status {
	case "", "all":
	case "ready", "pending_delete", "deleted", "failed":
		if status == "deleted" {
			query = query.Unscoped().Where("owner_user_id = ? AND status = ?", req.UserID, status)
		} else {
			query = query.Where("status = ?", status)
		}
	default:
		return nil, errors.New("invalid asset status")
	}

	kind := strings.TrimSpace(req.Kind)
	switch kind {
	case "", "all":
	case "image", "video", "file":
		query = query.Where("kind = ?", kind)
	default:
		return nil, errors.New("invalid asset kind")
	}

	if q := strings.TrimSpace(req.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where("filename LIKE ?", like)
	}

	var assets []models.Asset
	if err := query.
		Order("created_at desc").
		Limit(limit).
		Find(&assets).Error; err != nil {
		return nil, err
	}

	items := make([]AssetListItem, 0, len(assets))
	for _, asset := range assets {
		items = append(items, AssetListItem{
			ID:             asset.ID,
			Kind:           asset.Kind,
			Filename:       asset.Filename,
			MimeType:       asset.MimeType,
			FileSize:       asset.FileSize,
			Visibility:     asset.Visibility,
			Status:         asset.Status,
			ReferenceCount: asset.ReferenceCount,
			Deletable:      asset.ReferenceCount == 0 && asset.Status != "deleted",
			DocumentID:     asset.DocumentID,
			CreatedAt:      asset.CreatedAt,
			UpdatedAt:      asset.UpdatedAt,
		})
	}

	return &ListAssetsResult{Items: items}, nil
}

func GetAssetByID(assetID uuid.UUID) (*models.Asset, error) {
	var asset models.Asset
	result := database.DB.Where("id = ? AND deleted_at IS NULL", assetID).First(&asset)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("资源不存在")
		}
		return nil, result.Error
	}
	return &asset, nil
}

func UploadDocumentAsset(ctx context.Context, req UploadAssetRequest) (*UploadAssetResult, error) {
	log.Printf("[media.upload.service] validating document=%s user=%s", req.DocumentID, req.UserID)
	if req.FileHeader == nil {
		return nil, errors.New("file is required")
	}
	contentType, ok := normalizeAllowedContentType(
		strings.TrimSpace(req.FileHeader.Header.Get("Content-Type")),
		req.FileHeader.Filename,
	)
	if !ok {
		return nil, fmt.Errorf("unsupported file type: %s", contentType)
	}
	if err := ValidateVisibility(req.Visibility); err != nil {
		return nil, err
	}
	if err := ensureDocumentOwnership(req.UserID, req.DocumentID); err != nil {
		return nil, err
	}
	if err := initStorageProvider(); err != nil {
		return nil, err
	}

	file, err := req.FileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	log.Printf("[media.upload.service] reading file filename=%q size=%d", req.FileHeader.Filename, req.FileHeader.Size)
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	fileHash := computeFileHash(fileBytes)
	log.Printf("[media.upload.service] file hash computed hash=%s size=%d", fileHash, req.FileHeader.Size)

	if existing, err := findDeduplicatedAsset(req.UserID, fileHash, req.FileHeader.Size); err != nil {
		return nil, err
	} else if existing != nil {
		log.Printf("[media.upload.service] deduplicated existing asset=%s", existing.ID)
		return &UploadAssetResult{Asset: existing}, nil
	}

	objectKey := buildObjectKey(req.UserID, req.FileHeader.Filename)
	log.Printf("[media.upload.service] putting object key=%q provider=%T", objectKey, storageProvider)
	uploadResult, err := storageProvider.PutObject(ctx, PutObjectInput{
		ObjectKey:   objectKey,
		ContentType: contentType,
		Body:        bytes.NewReader(fileBytes),
	})
	if err != nil {
		return nil, err
	}
	log.Printf("[media.upload.service] object stored bucket=%q provider=%q", uploadResult.Bucket, uploadResult.Provider)

	kind := detectKind(req.FileHeader.Header.Get("Content-Type"))
	visibility := req.Visibility
	if visibility == "" {
		visibility = "private"
	}

	asset := &models.Asset{
		ID:              uuid.New(),
		OwnerUserID:     req.UserID,
		DocumentID:      &req.DocumentID,
		Kind:            kind,
		Filename:        req.FileHeader.Filename,
		FileHash:        fileHash,
		FileSize:        req.FileHeader.Size,
		MimeType:        contentType,
		StorageProvider: uploadResult.Provider,
		Bucket:          uploadResult.Bucket,
		ObjectKey:       objectKey,
		URL:             uploadResult.URL,
		Visibility:      visibility,
		Status:          "ready",
		ReferenceCount:  0,
		CreatedBy:       req.UserID,
	}

	if err := database.DB.Create(asset).Error; err != nil {
		return nil, err
	}
	log.Printf("[media.upload.service] asset row created asset=%s", asset.ID)
	return &UploadAssetResult{Asset: asset}, nil
}

func normalizeAllowedContentType(contentType string, filename string) (string, bool) {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if _, ok := allowedAssetMimeTypes[contentType]; ok {
		return contentType, true
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if fallbackType, ok := allowedAssetExtensions[ext]; ok {
		return fallbackType, true
	}

	if contentType != "" {
		return contentType, false
	}
	return ext, false
}

func ensureDocumentOwnership(userID, documentID uuid.UUID) error {
	var count int64
	if err := database.DB.Model(&models.Document{}).
		Where("id = ? AND owner_user_id = ? AND deleted_at IS NULL", documentID, userID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("文档不存在或无权访问")
	}
	return nil
}

func computeFileHash(fileBytes []byte) string {
	hasher := sha256.New()
	_, _ = hasher.Write(fileBytes)
	return hex.EncodeToString(hasher.Sum(nil))
}

func findDeduplicatedAsset(userID uuid.UUID, fileHash string, fileSize int64) (*models.Asset, error) {
	var asset models.Asset
	result := database.DB.
		Where("owner_user_id = ? AND file_hash = ? AND file_size = ? AND deleted_at IS NULL", userID, fileHash, fileSize).
		First(&asset)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &asset, nil
}

func GetOwnedAssetReferences(userID, assetID uuid.UUID) (*AssetReferencesResult, error) {
	asset, err := GetOwnedAsset(userID, assetID)
	if err != nil {
		return nil, err
	}

	var refs []models.DocumentAssetRef
	if err := database.DB.
		Where("owner_user_id = ? AND asset_id = ? AND ref_type = ?", userID, assetID, "editor_content").
		Find(&refs).Error; err != nil {
		return nil, err
	}

	documents := make([]AssetReferenceDocument, 0, len(refs))
	if len(refs) > 0 {
		documentIDs := make([]uuid.UUID, 0, len(refs))
		for _, ref := range refs {
			documentIDs = append(documentIDs, ref.DocumentID)
		}

		var docs []models.Document
		if err := database.DB.
			Where("owner_user_id = ? AND id IN ? AND deleted_at IS NULL", userID, documentIDs).
			Find(&docs).Error; err != nil {
			return nil, err
		}

		docByID := make(map[uuid.UUID]models.Document, len(docs))
		for _, doc := range docs {
			docByID[doc.ID] = doc
		}

		for _, ref := range refs {
			doc, ok := docByID[ref.DocumentID]
			if !ok {
				continue
			}
			documents = append(documents, AssetReferenceDocument{
				DocumentID: doc.ID,
				Title:      doc.Title,
				UpdatedAt:  doc.UpdatedAt,
			})
		}
	}

	sort.Slice(documents, func(i, j int) bool {
		return documents[i].UpdatedAt.After(documents[j].UpdatedAt)
	})

	return &AssetReferencesResult{
		AssetID:        asset.ID,
		ReferenceCount: len(documents),
		Documents:      documents,
	}, nil
}

func DeleteOwnedUnusedAsset(ctx context.Context, userID, assetID uuid.UUID) error {
	asset, err := GetOwnedAsset(userID, assetID)
	if err != nil {
		return err
	}
	if asset.ReferenceCount > 0 {
		return errors.New("asset is still referenced by documents")
	}
	if asset.Status == "deleted" {
		return errors.New("asset already deleted")
	}
	if err := initStorageProvider(); err != nil {
		return err
	}

	now := time.Now()
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var refCount int64
		if err := tx.Model(&models.DocumentAssetRef{}).
			Where("asset_id = ? AND ref_type = ?", assetID, "editor_content").
			Count(&refCount).Error; err != nil {
			return err
		}
		if refCount > 0 {
			return errors.New("asset is still referenced by documents")
		}

		if err := storageProvider.DeleteObject(ctx, asset.ObjectKey); err != nil {
			return err
		}

		if err := tx.Model(&models.Asset{}).
			Where("id = ? AND owner_user_id = ? AND deleted_at IS NULL", assetID, userID).
			Updates(map[string]any{
				"status":          "deleted",
				"reference_count": 0,
				"updated_at":      now,
				"deleted_at":      now,
			}).Error; err != nil {
			return err
		}

		return tx.Model(&models.AssetGCJob{}).
			Where("asset_id = ? AND status = ?", assetID, "pending").
			Updates(map[string]any{
				"status":     "cancelled",
				"updated_at": now,
			}).Error
	})
}

func buildObjectKey(userID uuid.UUID, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	day := time.Now().UTC().Format("20060102")
	return fmt.Sprintf("%s/%s/%s%s", userID.String(), day, uuid.NewString(), ext)
}

func detectKind(contentType string) string {
	if strings.HasPrefix(contentType, "image/") {
		return "image"
	}
	if strings.HasPrefix(contentType, "video/") {
		return "video"
	}
	return "file"
}
