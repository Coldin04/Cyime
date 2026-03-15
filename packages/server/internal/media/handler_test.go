package media

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func newMediaTestApp(userID uuid.UUID) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userId", userID.String())
		return c.Next()
	})
	app.Get("/media/assets", ListAssetsHandler)
	app.Get("/media/assets/:id/references", GetAssetReferencesHandler)
	app.Delete("/media/assets/:id", DeleteAssetHandler)
	return app
}

func TestListAssetsHandler_InvalidKindReturnsBadRequest(t *testing.T) {
	setupMediaTestDB(t)
	userID := uuid.New()

	app := newMediaTestApp(userID)
	req := httptest.NewRequest(http.MethodGet, "/media/assets?kind=audio", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestListAssetsHandler_ReturnsOwnedFilteredAssets(t *testing.T) {
	db := setupMediaTestDB(t)
	userID := uuid.New()
	otherUserID := uuid.New()
	docID := seedOwnedDocument(t, db, userID)

	ownedAsset := models.Asset{
		ID:              uuid.New(),
		OwnerUserID:     userID,
		DocumentID:      &docID,
		Kind:            "image",
		Filename:        "poster.png",
		FileHash:        "hash-poster",
		FileSize:        15,
		MimeType:        "image/png",
		StorageProvider: "local",
		ObjectKey:       "owner/poster.png",
		URL:             "http://example.test/poster.png",
		Visibility:      "private",
		Status:          "ready",
		ReferenceCount:  0,
		CreatedBy:       userID,
	}
	otherAsset := models.Asset{
		ID:              uuid.New(),
		OwnerUserID:     otherUserID,
		Kind:            "image",
		Filename:        "other.png",
		FileHash:        "hash-other-handler",
		FileSize:        18,
		MimeType:        "image/png",
		StorageProvider: "local",
		ObjectKey:       "other/other.png",
		URL:             "http://example.test/other.png",
		Visibility:      "private",
		Status:          "ready",
		ReferenceCount:  0,
		CreatedBy:       otherUserID,
	}
	if err := db.Create(&ownedAsset).Error; err != nil {
		t.Fatalf("create owned asset: %v", err)
	}
	if err := db.Create(&otherAsset).Error; err != nil {
		t.Fatalf("create other asset: %v", err)
	}

	app := newMediaTestApp(userID)
	req := httptest.NewRequest(http.MethodGet, "/media/assets?kind=image&status=ready&q=poster&limit=10", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload AssetListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
	if payload.Total != 1 || payload.HasMore {
		t.Fatalf("expected total=1 and hasMore=false, got total=%d hasMore=%v", payload.Total, payload.HasMore)
	}
	item := payload.Items[0]
	if item.ID != ownedAsset.ID || item.Filename != "poster.png" {
		t.Fatalf("unexpected item: %+v", item)
	}
	if !item.Deletable || item.ReferenceCount != 0 {
		t.Fatalf("expected deletable unreferenced item, got %+v", item)
	}
}

func TestGetAssetReferencesHandler_ReturnsOwnedReferences(t *testing.T) {
	db := setupMediaTestDB(t)
	userID := uuid.New()
	docID := seedOwnedDocument(t, db, userID)

	asset := models.Asset{
		ID:              uuid.New(),
		OwnerUserID:     userID,
		DocumentID:      &docID,
		Kind:            "image",
		Filename:        "cover.png",
		FileHash:        "hash-cover-ref",
		FileSize:        33,
		MimeType:        "image/png",
		StorageProvider: "local",
		ObjectKey:       "owner/cover.png",
		URL:             "http://example.test/cover.png",
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

	app := newMediaTestApp(userID)
	req := httptest.NewRequest(http.MethodGet, "/media/assets/"+asset.ID.String()+"/references", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload AssetReferencesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.AssetID != asset.ID || payload.ReferenceCount != 1 {
		t.Fatalf("unexpected payload summary: %+v", payload)
	}
	if len(payload.Documents) != 1 || payload.Documents[0].DocumentID != docID {
		t.Fatalf("unexpected referenced documents: %+v", payload.Documents)
	}
}

func TestDeleteAssetHandler_RejectsReferencedAsset(t *testing.T) {
	db := setupMediaTestDB(t)
	userID := uuid.New()
	docID := seedOwnedDocument(t, db, userID)

	asset := models.Asset{
		ID:              uuid.New(),
		OwnerUserID:     userID,
		DocumentID:      &docID,
		Kind:            "image",
		Filename:        "used.png",
		FileHash:        "hash-used-handler",
		FileSize:        14,
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

	app := newMediaTestApp(userID)
	req := httptest.NewRequest(http.MethodDelete, "/media/assets/"+asset.ID.String(), nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestDeleteAssetHandler_DeletesUnusedAsset(t *testing.T) {
	db := setupMediaTestDB(t)
	userID := uuid.New()
	docID := seedOwnedDocument(t, db, userID)

	rootDir := t.TempDir()
	t.Setenv("MEDIA_STORAGE_PROVIDER", "local")
	t.Setenv("MEDIA_LOCAL_ROOT_DIR", rootDir)
	storageProvider = nil
	t.Cleanup(func() { storageProvider = nil })

	objectKey := "owner/deletable.png"
	filePath := filepath.Join(rootDir, "owner", "deletable.png")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("abc"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	asset := models.Asset{
		ID:              uuid.New(),
		OwnerUserID:     userID,
		DocumentID:      &docID,
		Kind:            "image",
		Filename:        "deletable.png",
		FileHash:        "hash-deletable-handler",
		FileSize:        3,
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

	app := newMediaTestApp(userID)
	req := httptest.NewRequest(http.MethodDelete, "/media/assets/"+asset.ID.String(), nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected storage object deleted, stat err=%v", err)
	}

	var got models.Asset
	if err := db.Unscoped().First(&got, "id = ?", asset.ID).Error; err != nil {
		t.Fatalf("load deleted asset: %v", err)
	}
	if got.Status != "deleted" || !got.DeletedAt.Valid {
		t.Fatalf("expected deleted asset row, got status=%s deletedAt=%v", got.Status, got.DeletedAt.Valid)
	}
}
