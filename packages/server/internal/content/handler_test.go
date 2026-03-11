package content

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func newContentTestApp(userID uuid.UUID) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userId", userID.String())
		return c.Next()
	})
	app.Get("/documents/:id/content", GetContentHandler)
	app.Put("/documents/:id/content", UpdateContentHandler)
	return app
}

func TestGetContentHandler_CrossUserDenied(t *testing.T) {
	db := setupContentTestDB(t)
	ownerID := uuid.New()
	attackerID := uuid.New()
	docID, _ := seedDocumentForContent(t, db, ownerID, "owner-doc", "secret")

	app := newContentTestApp(attackerID)
	req := httptest.NewRequest(http.MethodGet, "/documents/"+docID.String()+"/content", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateContentHandler_CrossUserDeniedAndDataUnchanged(t *testing.T) {
	db := setupContentTestDB(t)
	ownerID := uuid.New()
	attackerID := uuid.New()
	docID, contentID := seedDocumentForContent(t, db, ownerID, "owner-doc", "before")

	app := newContentTestApp(attackerID)
	body := bytes.NewBufferString(`{"content":"hacked"}`)
	req := httptest.NewRequest(http.MethodPut, "/documents/"+docID.String()+"/content", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var got models.DocumentContent
	if err := db.First(&got, "id = ?", contentID).Error; err != nil {
		t.Fatalf("load content: %v", err)
	}
	if got.ContentMarkdown != "before" {
		t.Fatalf("expected content unchanged, got %q", got.ContentMarkdown)
	}
}

