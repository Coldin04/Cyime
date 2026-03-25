package media

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"testing"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupMediaTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.UserImageBedConfig{}, &models.Document{}, &models.Asset{}, &models.DocumentAssetRef{}, &models.AssetGCJob{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	database.DB = db
	return db
}

func seedOwnedDocument(t *testing.T, db *gorm.DB, userID uuid.UUID) uuid.UUID {
	return seedOwnedDocumentWithImageTarget(t, db, userID, "")
}

func seedOwnedDocumentWithImageTarget(t *testing.T, db *gorm.DB, userID uuid.UUID, preferredImageTargetID string) uuid.UUID {
	t.Helper()
	doc := models.Document{
		ID:                     uuid.New(),
		OwnerUserID:            userID,
		Title:                  "doc",
		DocumentType:           "rich_text",
		PreferredImageTargetID: preferredImageTargetID,
		EditorType:             "tiptap",
		CreatedBy:              userID,
		UpdatedBy:              userID,
	}
	if err := db.Create(&doc).Error; err != nil {
		t.Fatalf("create document: %v", err)
	}
	return doc.ID
}

func makeFileHeader(t *testing.T, fieldName, filename string, content []byte) *multipart.FileHeader {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(8 << 20); err != nil {
		t.Fatalf("parse multipart form: %v", err)
	}
	return req.MultipartForm.File[fieldName][0]
}

func TestUploadDocumentAsset_DeduplicatesByHashAndSize(t *testing.T) {
	db := setupMediaTestDB(t)
	userID := uuid.New()
	docID := seedOwnedDocument(t, db, userID)

	t.Setenv("MEDIA_STORAGE_PROVIDER", "local")
	t.Setenv("MEDIA_LOCAL_ROOT_DIR", t.TempDir())
	storageProvider = nil

	headerA := makeFileHeader(t, "file", "photo.png", []byte("same-content"))
	first, err := UploadDocumentAsset(context.Background(), UploadAssetRequest{
		DocumentID: docID,
		UserID:     userID,
		FileHeader: headerA,
		Visibility: "private",
	})
	if err != nil {
		t.Fatalf("first upload: %v", err)
	}

	headerB := makeFileHeader(t, "file", "duplicate.png", []byte("same-content"))
	second, err := UploadDocumentAsset(context.Background(), UploadAssetRequest{
		DocumentID: docID,
		UserID:     userID,
		FileHeader: headerB,
		Visibility: "private",
	})
	if err != nil {
		t.Fatalf("second upload: %v", err)
	}

	if first.Asset.ID != second.Asset.ID {
		t.Fatalf("expected deduplicated asset id, got %s and %s", first.Asset.ID, second.Asset.ID)
	}

	var asset models.Asset
	if err := db.First(&asset, "id = ?", first.Asset.ID).Error; err != nil {
		t.Fatalf("load asset: %v", err)
	}
	if asset.ReferenceCount != 0 {
		t.Fatalf("expected reference_count=0 before save-time sync, got %d", asset.ReferenceCount)
	}
}

func TestGetOwnedAssetReferences_ReturnsReferencingDocuments(t *testing.T) {
	db := setupMediaTestDB(t)
	userID := uuid.New()
	docID := seedOwnedDocument(t, db, userID)

	asset := models.Asset{
		ID:              uuid.New(),
		OwnerUserID:     userID,
		DocumentID:      &docID,
		Filename:        "photo.png",
		FileHash:        "hash-photo",
		FileSize:        12,
		MimeType:        "image/png",
		StorageProvider: "local",
		ObjectKey:       "owner/photo.png",
		URL:             "http://example.test/photo.png",
		Visibility:      "private",
		Status:          "ready",
		ReferenceCount:  1,
		CreatedBy:       userID,
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := db.Create(&models.DocumentAssetRef{
		ID:          uuid.New(),
		DocumentID:  docID,
		AssetID:     asset.ID,
		OwnerUserID: userID,
		RefType:     "editor_content",
	}).Error; err != nil {
		t.Fatalf("create ref: %v", err)
	}

	result, err := GetOwnedAssetReferences(userID, asset.ID)
	if err != nil {
		t.Fatalf("get references: %v", err)
	}
	if result.ReferenceCount != 1 {
		t.Fatalf("expected referenceCount=1, got %d", result.ReferenceCount)
	}
	if len(result.Documents) != 1 || result.Documents[0].DocumentID != docID {
		t.Fatalf("unexpected documents: %+v", result.Documents)
	}
}

func TestDeleteOwnedUnusedAsset_DeletesStorageAndMarksDeleted(t *testing.T) {
	db := setupMediaTestDB(t)
	userID := uuid.New()
	docID := seedOwnedDocument(t, db, userID)

	rootDir := t.TempDir()
	t.Setenv("MEDIA_STORAGE_PROVIDER", "local")
	t.Setenv("MEDIA_LOCAL_ROOT_DIR", rootDir)
	storageProvider = nil

	objectKey := "owner/deletable.png"
	filePath := rootDir + string(os.PathSeparator) + "owner" + string(os.PathSeparator) + "deletable.png"
	if err := os.MkdirAll(rootDir+string(os.PathSeparator)+"owner", 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	asset := models.Asset{
		ID:              uuid.New(),
		OwnerUserID:     userID,
		DocumentID:      &docID,
		Filename:        "deletable.png",
		FileHash:        "hash-delete",
		FileSize:        4,
		MimeType:        "image/png",
		StorageProvider: "local",
		ObjectKey:       objectKey,
		URL:             "http://example.test/deletable.png",
		Visibility:      "private",
		Status:          "pending_delete",
		ReferenceCount:  0,
		CreatedBy:       userID,
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := db.Create(&models.AssetGCJob{
		ID:       uuid.New(),
		AssetID:  asset.ID,
		JobType:  "delete",
		Status:   "pending",
		RunAfter: asset.CreatedAt,
	}).Error; err != nil {
		t.Fatalf("create gc job: %v", err)
	}

	if err := DeleteOwnedUnusedAsset(context.Background(), userID, asset.ID); err != nil {
		t.Fatalf("delete unused asset: %v", err)
	}

	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected storage object deleted, stat err=%v", err)
	}

	var got models.Asset
	if err := db.Unscoped().First(&got, "id = ?", asset.ID).Error; err != nil {
		t.Fatalf("load deleted asset: %v", err)
	}
	if got.Status != "deleted" || got.DeletedAt.Valid == false {
		t.Fatalf("expected deleted asset row, got status=%s deletedAt=%v", got.Status, got.DeletedAt.Valid)
	}

	var job models.AssetGCJob
	if err := db.First(&job, "asset_id = ?", asset.ID).Error; err != nil {
		t.Fatalf("load gc job: %v", err)
	}
	if job.Status != "cancelled" {
		t.Fatalf("expected gc job cancelled, got %s", job.Status)
	}
}

func TestDeleteOwnedUnusedAsset_RejectsReferencedAsset(t *testing.T) {
	db := setupMediaTestDB(t)
	userID := uuid.New()
	docID := seedOwnedDocument(t, db, userID)

	asset := models.Asset{
		ID:              uuid.New(),
		OwnerUserID:     userID,
		DocumentID:      &docID,
		Filename:        "used.png",
		FileHash:        "hash-used",
		FileSize:        4,
		MimeType:        "image/png",
		StorageProvider: "local",
		ObjectKey:       "owner/used.png",
		URL:             "http://example.test/used.png",
		Visibility:      "private",
		Status:          "ready",
		ReferenceCount:  1,
		CreatedBy:       userID,
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := db.Create(&models.DocumentAssetRef{
		ID:          uuid.New(),
		DocumentID:  docID,
		AssetID:     asset.ID,
		OwnerUserID: userID,
		RefType:     "editor_content",
	}).Error; err != nil {
		t.Fatalf("create ref: %v", err)
	}

	if err := DeleteOwnedUnusedAsset(context.Background(), userID, asset.ID); err == nil || err.Error() != "asset is still referenced by documents" {
		t.Fatalf("expected referenced asset rejection, got %v", err)
	}
}

func TestListOwnedAssets_FiltersAndMarksDeletable(t *testing.T) {
	db := setupMediaTestDB(t)
	userID := uuid.New()
	otherUserID := uuid.New()
	docID := seedOwnedDocument(t, db, userID)

	assets := []models.Asset{
		{
			ID:              uuid.New(),
			OwnerUserID:     userID,
			DocumentID:      &docID,
			Kind:            "image",
			Filename:        "cover.png",
			FileHash:        "hash-cover",
			FileSize:        10,
			MimeType:        "image/png",
			StorageProvider: "local",
			ObjectKey:       "owner/cover.png",
			URL:             "http://example.test/cover.png",
			Visibility:      "private",
			Status:          "ready",
			ReferenceCount:  1,
			CreatedBy:       userID,
		},
		{
			ID:              uuid.New(),
			OwnerUserID:     userID,
			DocumentID:      &docID,
			Kind:            "video",
			Filename:        "clip.webm",
			FileHash:        "hash-clip",
			FileSize:        20,
			MimeType:        "video/webm",
			StorageProvider: "local",
			ObjectKey:       "owner/clip.webm",
			URL:             "http://example.test/clip.webm",
			Visibility:      "private",
			Status:          "pending_delete",
			ReferenceCount:  0,
			CreatedBy:       userID,
		},
		{
			ID:              uuid.New(),
			OwnerUserID:     otherUserID,
			Kind:            "image",
			Filename:        "other.png",
			FileHash:        "hash-other",
			FileSize:        30,
			MimeType:        "image/png",
			StorageProvider: "local",
			ObjectKey:       "other/other.png",
			URL:             "http://example.test/other.png",
			Visibility:      "private",
			Status:          "ready",
			ReferenceCount:  0,
			CreatedBy:       otherUserID,
		},
	}
	for _, asset := range assets {
		if err := db.Create(&asset).Error; err != nil {
			t.Fatalf("create asset %s: %v", asset.Filename, err)
		}
	}

	result, err := ListOwnedAssets(ListAssetsRequest{
		UserID: userID,
		Kind:   "video",
		Status: "pending_delete",
		Query:  "clip",
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}

	if len(result.Items) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(result.Items))
	}
	item := result.Items[0]
	if item.Filename != "clip.webm" || item.Kind != "video" || item.Status != "pending_delete" {
		t.Fatalf("unexpected listed asset: %+v", item)
	}
	if !item.Deletable || item.ReferenceCount != 0 {
		t.Fatalf("expected pending_delete asset to be deletable with ref=0, got %+v", item)
	}
	if result.Total != 1 || result.HasMore {
		t.Fatalf("expected total=1 and hasMore=false, got total=%d hasMore=%v", result.Total, result.HasMore)
	}
}

func TestListOwnedAssets_IncludesDeletedWhenRequested(t *testing.T) {
	db := setupMediaTestDB(t)
	userID := uuid.New()
	asset := models.Asset{
		ID:              uuid.New(),
		OwnerUserID:     userID,
		Kind:            "image",
		Filename:        "deleted.png",
		FileHash:        "hash-deleted",
		FileSize:        10,
		MimeType:        "image/png",
		StorageProvider: "local",
		ObjectKey:       "owner/deleted.png",
		URL:             "http://example.test/deleted.png",
		Visibility:      "private",
		Status:          "deleted",
		ReferenceCount:  0,
		CreatedBy:       userID,
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := db.Delete(&asset).Error; err != nil {
		t.Fatalf("soft delete asset: %v", err)
	}

	result, err := ListOwnedAssets(ListAssetsRequest{
		UserID: userID,
		Status: "deleted",
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("list deleted assets: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Status != "deleted" {
		t.Fatalf("unexpected deleted asset list: %+v", result.Items)
	}
	if result.Items[0].Deletable {
		t.Fatalf("deleted asset should not be deletable again")
	}
}

func TestListOwnedAssets_PaginatesLikeWorkspaceList(t *testing.T) {
	db := setupMediaTestDB(t)
	userID := uuid.New()

	assets := []models.Asset{
		{
			ID:              uuid.New(),
			OwnerUserID:     userID,
			Kind:            "image",
			Filename:        "a.png",
			FileHash:        "hash-a-page",
			FileSize:        1,
			MimeType:        "image/png",
			StorageProvider: "local",
			ObjectKey:       "owner/a.png",
			URL:             "http://example.test/a.png",
			Visibility:      "private",
			Status:          "ready",
			CreatedBy:       userID,
		},
		{
			ID:              uuid.New(),
			OwnerUserID:     userID,
			Kind:            "image",
			Filename:        "b.png",
			FileHash:        "hash-b-page",
			FileSize:        1,
			MimeType:        "image/png",
			StorageProvider: "local",
			ObjectKey:       "owner/b.png",
			URL:             "http://example.test/b.png",
			Visibility:      "private",
			Status:          "ready",
			CreatedBy:       userID,
		},
		{
			ID:              uuid.New(),
			OwnerUserID:     userID,
			Kind:            "image",
			Filename:        "c.png",
			FileHash:        "hash-c-page",
			FileSize:        1,
			MimeType:        "image/png",
			StorageProvider: "local",
			ObjectKey:       "owner/c.png",
			URL:             "http://example.test/c.png",
			Visibility:      "private",
			Status:          "ready",
			CreatedBy:       userID,
		},
	}
	for _, asset := range assets {
		if err := db.Create(&asset).Error; err != nil {
			t.Fatalf("create asset %s: %v", asset.Filename, err)
		}
	}

	result, err := ListOwnedAssets(ListAssetsRequest{
		UserID: userID,
		Limit:  2,
		Offset: 1,
	})
	if err != nil {
		t.Fatalf("list assets with pagination: %v", err)
	}

	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Total != 3 {
		t.Fatalf("expected total=3, got %d", result.Total)
	}
	if result.HasMore {
		t.Fatalf("expected hasMore=false at offset 1 limit 2, got true")
	}
}
