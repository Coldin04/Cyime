package content

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"g.co1d.in/Coldin04/Cyime/server/internal/models"
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

func TestContentHandlers_OwnerHTTPSaveRoundTrip(t *testing.T) {
	t.Setenv("COLLABORATION_ENABLED", "false")

	db := setupContentTestDB(t)
	ownerID := uuid.New()
	docID, _ := seedDocumentForContent(t, db, ownerID, "owner-doc", `{"type":"doc","content":[{"type":"paragraph"}]}`)
	app := newContentTestApp(ownerID)

	wantContent := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"saved over HTTP"}]}]}`
	updateBody := bytes.NewBufferString(`{"contentJson":` + wantContent + `}`)
	updateReq := httptest.NewRequest(http.MethodPut, "/documents/"+docID.String()+"/content", updateBody)
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, err := app.Test(updateReq, -1)
	if err != nil {
		t.Fatalf("update request failed: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update status 200, got %d", updateResp.StatusCode)
	}

	var updateResult UpdateContentResult
	if err := json.NewDecoder(updateResp.Body).Decode(&updateResult); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if !updateResult.Success || updateResult.ContentVersion != 2 {
		t.Fatalf("unexpected update response: %+v", updateResult)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/documents/"+docID.String()+"/content", nil)
	getResp, err := app.Test(getReq, -1)
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get status 200, got %d", getResp.StatusCode)
	}

	var getResult GetContentResult
	if err := json.NewDecoder(getResp.Body).Decode(&getResult); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if string(getResult.ContentJSON) != wantContent || getResult.ContentVersion != 2 {
		t.Fatalf("unexpected persisted content: %+v", getResult)
	}
}

func TestGetContentHandler_CrossUserDenied(t *testing.T) {
	db := setupContentTestDB(t)
	ownerID := uuid.New()
	attackerID := uuid.New()
	docID, _ := seedDocumentForContent(t, db, ownerID, "owner-doc", `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"secret"}]}]}`)

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
	docID, contentID := seedDocumentForContent(t, db, ownerID, "owner-doc", `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"before"}]}]}`)

	app := newContentTestApp(attackerID)
	body := bytes.NewBufferString(`{"contentJson":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hacked"}]}]}}`)
	req := httptest.NewRequest(http.MethodPut, "/documents/"+docID.String()+"/content", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var got models.DocumentBody
	if err := db.First(&got, "id = ?", contentID).Error; err != nil {
		t.Fatalf("load content: %v", err)
	}
	expected := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"before"}]}]}`
	if got.ContentJSON != expected {
		t.Fatalf("expected content unchanged, got %q", got.ContentJSON)
	}
}

func TestUpdateContentHandler_WorkspaceQuotaExceeded(t *testing.T) {
	db := setupContentTestDB(t)
	ownerID := uuid.New()
	docID, _ := seedDocumentForContent(t, db, ownerID, "owner-doc", `{"type":"doc","content":[{"type":"paragraph"}]}`)

	largeDescription := strings.Repeat("d", maxWorkspaceStorageBytesPerUser-40)
	folder := models.Folder{
		ID:          uuid.New(),
		OwnerUserID: ownerID,
		Name:        "large",
		Description: &largeDescription,
		CreatedBy:   ownerID,
		UpdatedBy:   ownerID,
	}
	if err := db.Create(&folder).Error; err != nil {
		t.Fatalf("create folder: %v", err)
	}

	app := newContentTestApp(ownerID)
	body := bytes.NewBufferString(`{"contentJson":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"this update should exceed workspace quota"}]}]}}`)
	req := httptest.NewRequest(http.MethodPut, "/documents/"+docID.String()+"/content", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	var payload ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Message != ErrWorkspaceStorageQuotaExceeded.Error() {
		t.Fatalf("expected quota message, got %q", payload.Message)
	}
}
