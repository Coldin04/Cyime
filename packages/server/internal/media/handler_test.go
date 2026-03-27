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
	app.Get("/media/shared-assets", ListSharedAssetsHandler)
	app.Get("/media/assets/:id/url", GetAssetURLHandler)
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

	ownedBlob := seedBlob(t, db, "owner/poster.png", "image/png", 15, "hash-poster")
	ownedAsset := models.Asset{
		ID:             uuid.New(),
		OwnerUserID:    userID,
		DocumentID:     &docID,
		BlobID:         ownedBlob.ID,
		Kind:           "image",
		Filename:       "poster.png",
		URL:            ownedBlob.URL,
		Visibility:     "private",
		Status:         "ready",
		ReferenceCount: 0,
		CreatedBy:      userID,
	}
	otherBlob := seedBlob(t, db, "other/other.png", "image/png", 18, "hash-other-handler")
	otherAsset := models.Asset{
		ID:             uuid.New(),
		OwnerUserID:    otherUserID,
		BlobID:         otherBlob.ID,
		Kind:           "image",
		Filename:       "other.png",
		URL:            otherBlob.URL,
		Visibility:     "private",
		Status:         "ready",
		ReferenceCount: 0,
		CreatedBy:      otherUserID,
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

	blob := seedBlob(t, db, "owner/cover.png", "image/png", 33, "hash-cover-ref")
	asset := models.Asset{
		ID:             uuid.New(),
		OwnerUserID:    userID,
		DocumentID:     &docID,
		BlobID:         blob.ID,
		Kind:           "image",
		Filename:       "cover.png",
		URL:            blob.URL,
		Visibility:     "private",
		Status:         "ready",
		ReferenceCount: 1,
		CreatedBy:      userID,
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

	blob := seedBlob(t, db, "owner/used.png", "image/png", 14, "hash-used-handler")
	asset := models.Asset{
		ID:             uuid.New(),
		OwnerUserID:    userID,
		DocumentID:     &docID,
		BlobID:         blob.ID,
		Kind:           "image",
		Filename:       "used.png",
		URL:            blob.URL,
		Visibility:     "private",
		Status:         "ready",
		ReferenceCount: 1,
		CreatedBy:      userID,
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

func TestGetAssetURLHandler_AllowsSharedViewerAccess(t *testing.T) {
	db := setupMediaTestDB(t)
	ownerID := uuid.New()
	viewerID := uuid.New()
	docID := seedOwnedDocument(t, db, ownerID)
	seedDocumentPermission(t, db, docID, viewerID, ownerID, "viewer")
	t.Setenv("MEDIA_TOKEN_SECRET", "test-media-secret")

	blob := seedBlob(t, db, "owner/shared.png", "image/png", 14, "hash-shared-url")
	asset := models.Asset{
		ID:             uuid.New(),
		OwnerUserID:    ownerID,
		DocumentID:     &docID,
		BlobID:         blob.ID,
		Kind:           "image",
		Filename:       "shared.png",
		URL:            blob.URL,
		Visibility:     "private",
		Status:         "ready",
		ReferenceCount: 1,
		CreatedBy:      ownerID,
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := db.Create(&models.DocumentAssetRef{
		ID:          uuid.New(),
		DocumentID:  docID,
		AssetID:     asset.ID,
		OwnerUserID: ownerID,
		RefType:     "editor_content",
	}).Error; err != nil {
		t.Fatalf("create ref: %v", err)
	}

	app := newMediaTestApp(viewerID)
	req := httptest.NewRequest(http.MethodGet, "/media/assets/"+asset.ID.String()+"/url", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestListSharedAssetsHandler_ReturnsSharedEditorAssetsOnly(t *testing.T) {
	db := setupMediaTestDB(t)
	ownerID := uuid.New()
	editorID := uuid.New()
	viewerID := uuid.New()
	docID := seedOwnedDocument(t, db, ownerID)
	seedDocumentPermission(t, db, docID, editorID, ownerID, "editor")
	seedDocumentPermission(t, db, docID, viewerID, ownerID, "viewer")

	blob := seedBlob(t, db, "owner/shared-lib.png", "image/png", 21, "hash-shared-lib")
	asset := models.Asset{
		ID:             uuid.New(),
		OwnerUserID:    ownerID,
		DocumentID:     &docID,
		BlobID:         blob.ID,
		Kind:           "image",
		Filename:       "shared-lib.png",
		URL:            blob.URL,
		Visibility:     "private",
		Status:         "ready",
		ReferenceCount: 1,
		CreatedBy:      ownerID,
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := db.Create(&models.DocumentAssetRef{
		ID:          uuid.New(),
		DocumentID:  docID,
		AssetID:     asset.ID,
		OwnerUserID: ownerID,
		RefType:     "editor_content",
	}).Error; err != nil {
		t.Fatalf("create ref: %v", err)
	}

	editorApp := newMediaTestApp(editorID)
	editorReq := httptest.NewRequest(http.MethodGet, "/media/shared-assets", nil)
	editorResp, err := editorApp.Test(editorReq, -1)
	if err != nil {
		t.Fatalf("editor request failed: %v", err)
	}
	if editorResp.StatusCode != http.StatusOK {
		t.Fatalf("expected editor 200, got %d", editorResp.StatusCode)
	}
	var editorPayload SharedAssetListResponse
	if err := json.NewDecoder(editorResp.Body).Decode(&editorPayload); err != nil {
		t.Fatalf("decode editor response: %v", err)
	}
	if len(editorPayload.Items) != 1 || editorPayload.Items[0].ID != asset.ID {
		t.Fatalf("unexpected editor shared assets: %+v", editorPayload.Items)
	}
	if editorPayload.Items[0].DocumentCount != 1 || len(editorPayload.Items[0].Documents) != 1 {
		t.Fatalf("expected document linkage in shared asset payload: %+v", editorPayload.Items[0])
	}

	viewerApp := newMediaTestApp(viewerID)
	viewerReq := httptest.NewRequest(http.MethodGet, "/media/shared-assets", nil)
	viewerResp, err := viewerApp.Test(viewerReq, -1)
	if err != nil {
		t.Fatalf("viewer request failed: %v", err)
	}
	if viewerResp.StatusCode != http.StatusOK {
		t.Fatalf("expected viewer 200, got %d", viewerResp.StatusCode)
	}
	var viewerPayload SharedAssetListResponse
	if err := json.NewDecoder(viewerResp.Body).Decode(&viewerPayload); err != nil {
		t.Fatalf("decode viewer response: %v", err)
	}
	if len(viewerPayload.Items) != 0 {
		t.Fatalf("viewer should not receive shared editable assets, got %+v", viewerPayload.Items)
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

	blob := seedBlob(t, db, objectKey, "image/png", 3, "hash-deletable-handler")
	asset := models.Asset{
		ID:             uuid.New(),
		OwnerUserID:    userID,
		DocumentID:     &docID,
		BlobID:         blob.ID,
		Kind:           "image",
		Filename:       "deletable.png",
		URL:            blob.URL,
		Visibility:     "private",
		Status:         "pending_delete",
		ReferenceCount: 0,
		CreatedBy:      userID,
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

	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected storage object to remain until blob gc, stat err=%v", err)
	}

	var got models.Asset
	if err := db.Unscoped().First(&got, "id = ?", asset.ID).Error; err != nil {
		t.Fatalf("load deleted asset: %v", err)
	}
	if got.Status != "deleted" || !got.DeletedAt.Valid {
		t.Fatalf("expected deleted asset row, got status=%s deletedAt=%v", got.Status, got.DeletedAt.Valid)
	}

	var blobJob models.BlobGCJob
	if err := db.First(&blobJob, "blob_id = ?", blob.ID).Error; err != nil {
		t.Fatalf("load blob gc job: %v", err)
	}
	if blobJob.Status != "pending" {
		t.Fatalf("expected pending blob job, got %s", blobJob.Status)
	}
}
