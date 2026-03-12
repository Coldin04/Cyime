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
		if err := incrementReference(existing.ID); err != nil {
			return nil, err
		}
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
		ReferenceCount:  1,
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

func incrementReference(assetID uuid.UUID) error {
	return database.DB.Model(&models.Asset{}).
		Where("id = ?", assetID).
		Updates(map[string]any{
			"reference_count": gorm.Expr("reference_count + 1"),
			"updated_at":      time.Now(),
		}).Error
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
