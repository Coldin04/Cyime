package media

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
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
	if err := db.AutoMigrate(&models.User{}, &models.Document{}, &models.Asset{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	database.DB = db
	return db
}

func seedOwnedDocument(t *testing.T, db *gorm.DB, userID uuid.UUID) uuid.UUID {
	t.Helper()
	doc := models.Document{
		ID:           uuid.New(),
		OwnerUserID:  userID,
		Title:        "doc",
		DocumentType: "rich_text",
		EditorType:   "tiptap",
		CreatedBy:    userID,
		UpdatedBy:    userID,
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
	if asset.ReferenceCount != 2 {
		t.Fatalf("expected reference_count=2, got %d", asset.ReferenceCount)
	}
}
