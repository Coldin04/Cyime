package workspace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func newRealtimeStateTestApp(userID uuid.UUID) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userId", userID.String())
		return c.Next()
	})
	app.Get("/realtime/documents/:id/state", GetYjsStateHandler)
	app.Put("/realtime/documents/:id/state", UpdateYjsStateHandler)
	return app
}

func putYjsState(
	t *testing.T,
	app *fiber.App,
	documentID uuid.UUID,
	yjsState string,
	yjsStateVector string,
	expectedVersion int64,
) *http.Response {
	t.Helper()
	payload := fmt.Sprintf(
		`{"yjsState":%q,"yjsStateVector":%q,"expectedYjsVersion":%d}`,
		yjsState, yjsStateVector, expectedVersion,
	)
	req := httptest.NewRequest(
		http.MethodPut,
		"/realtime/documents/"+documentID.String()+"/state",
		bytes.NewBufferString(payload),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

// TestUpdateYjsStateHandler_CreatesRowWhenMissing covers the silent-no-op
// half of P1-#4: the previous handler issued an UPDATE and ignored
// RowsAffected, so the very first save for a document returned 200 while
// writing nothing.
func TestUpdateYjsStateHandler_CreatesRowWhenMissing(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "yjs-doc")

	// Drop the body row that the seed helper created so we can exercise the
	// "no row exists" branch end-to-end.
	if err := db.Unscoped().Where("document_id = ?", docID).Delete(&models.DocumentBody{}).Error; err != nil {
		t.Fatalf("clear seed body: %v", err)
	}

	app := newRealtimeStateTestApp(ownerID)
	resp := putYjsState(t, app, docID, "AAEC", "AAA=", 0)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body GetYjsStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.YjsVersion != 1 {
		t.Fatalf("expected version 1 after initial create, got %d", body.YjsVersion)
	}

	var stored models.DocumentBody
	if err := db.Where("document_id = ?", docID).First(&stored).Error; err != nil {
		t.Fatalf("load stored body: %v", err)
	}
	if stored.YjsState != "AAEC" || stored.YjsStateVector != "AAA=" {
		t.Fatalf("stored body content mismatch: %+v", stored)
	}
	if stored.YjsVersion != 1 {
		t.Fatalf("stored YjsVersion = %d, want 1", stored.YjsVersion)
	}
}

// TestUpdateYjsStateHandler_BumpsVersionOnSuccess validates the optimistic
// concurrency happy path: caller echoes the current version and the row is
// updated with version+1.
func TestUpdateYjsStateHandler_BumpsVersionOnSuccess(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "yjs-doc")

	// Pin the body row to a known version.
	if err := db.Model(&models.DocumentBody{}).
		Where("document_id = ?", docID).
		Updates(map[string]any{"yjs_version": 7, "yjs_state": "OLD"}).Error; err != nil {
		t.Fatalf("seed version: %v", err)
	}

	app := newRealtimeStateTestApp(ownerID)
	resp := putYjsState(t, app, docID, "NEW", "VEC", 7)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body GetYjsStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.YjsVersion != 8 {
		t.Fatalf("expected version 8 after bump, got %d", body.YjsVersion)
	}

	var stored models.DocumentBody
	if err := db.Where("document_id = ?", docID).First(&stored).Error; err != nil {
		t.Fatalf("load stored body: %v", err)
	}
	if stored.YjsVersion != 8 {
		t.Fatalf("stored YjsVersion = %d, want 8", stored.YjsVersion)
	}
	if stored.YjsState != "NEW" {
		t.Fatalf("stored YjsState = %q, want NEW", stored.YjsState)
	}
}

// TestUpdateYjsStateHandler_RejectsStaleVersion is the security-critical
// test: a writer holding an old version cannot blindly overwrite fresher
// state. Without this check, the original handler would succeed and silently
// roll back another collaborator's edits.
func TestUpdateYjsStateHandler_RejectsStaleVersion(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "yjs-doc")

	if err := db.Model(&models.DocumentBody{}).
		Where("document_id = ?", docID).
		Updates(map[string]any{"yjs_version": 5, "yjs_state": "FRESH"}).Error; err != nil {
		t.Fatalf("seed version: %v", err)
	}

	app := newRealtimeStateTestApp(ownerID)
	resp := putYjsState(t, app, docID, "STALE_OVERWRITE", "VEC", 2)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 conflict, got %d", resp.StatusCode)
	}

	var conflict YjsStateConflictResponse
	if err := json.NewDecoder(resp.Body).Decode(&conflict); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if conflict.CurrentVersion != 5 {
		t.Fatalf("expected currentYjsVersion 5, got %d", conflict.CurrentVersion)
	}

	// Crucially, the stored row must be untouched.
	var stored models.DocumentBody
	if err := db.Where("document_id = ?", docID).First(&stored).Error; err != nil {
		t.Fatalf("load stored body: %v", err)
	}
	if stored.YjsState != "FRESH" {
		t.Fatalf("stale write leaked into storage: state = %q", stored.YjsState)
	}
	if stored.YjsVersion != 5 {
		t.Fatalf("stale write bumped version: %d", stored.YjsVersion)
	}
}

// TestGetYjsStateHandler_ReturnsVersion verifies the GET side now exposes
// yjsVersion so the realtime client can echo it on the next save.
func TestGetYjsStateHandler_ReturnsVersion(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "yjs-doc")

	if err := db.Model(&models.DocumentBody{}).
		Where("document_id = ?", docID).
		Updates(map[string]any{"yjs_version": 42, "yjs_state": "data", "yjs_state_vector": "vec"}).Error; err != nil {
		t.Fatalf("seed version: %v", err)
	}

	app := newRealtimeStateTestApp(ownerID)
	req := httptest.NewRequest(http.MethodGet, "/realtime/documents/"+docID.String()+"/state", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body GetYjsStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.YjsVersion != 42 {
		t.Fatalf("expected version 42, got %d", body.YjsVersion)
	}
	if body.YjsState != "data" || body.YjsStateVector != "vec" {
		t.Fatalf("unexpected payload: %+v", body)
	}
}

// TestUpdateYjsStateHandler_ViewerRoleDenied confirms the existing edit-ACL
// check still rejects viewers, so adding optimistic concurrency hasn't
// silently widened the write surface.
func TestUpdateYjsStateHandler_ViewerRoleDenied(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	viewerID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "yjs-doc")
	seedVerifiedUser(t, db, viewerID, "viewer@example.com")
	seedWorkspacePermission(t, db, docID, viewerID, ownerID, "viewer")

	app := newRealtimeStateTestApp(viewerID)
	resp := putYjsState(t, app, docID, "PWN", "VEC", 1)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for viewer write, got %d", resp.StatusCode)
	}

	// Storage must remain untouched.
	var stored models.DocumentBody
	if err := db.Where("document_id = ?", docID).First(&stored).Error; err != nil {
		t.Fatalf("load stored body: %v", err)
	}
	if stored.YjsState == "PWN" {
		t.Fatal("viewer write leaked into storage")
	}
}
