package media

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	item := payload.Items[0]
	if item.ID != ownedAsset.ID || item.Filename != "poster.png" {
		t.Fatalf("unexpected item: %+v", item)
	}
	if !item.Deletable || item.ReferenceCount != 0 {
		t.Fatalf("expected deletable unreferenced item, got %+v", item)
	}
}
