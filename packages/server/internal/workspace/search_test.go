package workspace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"g.co1d.in/Coldin04/Cyime/server/internal/models"
	"github.com/google/uuid"
)

func TestSearchWorkspace_ReturnsOwnedSharedAndMediaResults(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	searcherID := uuid.New()

	folder := models.Folder{
		ID:          uuid.New(),
		OwnerUserID: searcherID,
		Name:        "挂载资料",
		CreatedBy:   searcherID,
		UpdatedBy:   searcherID,
	}
	if err := db.Create(&folder).Error; err != nil {
		t.Fatalf("create folder: %v", err)
	}

	ownedDocID := seedDocumentForWorkspace(t, db, searcherID, "挂载手册")
	sharedDocID := seedDocumentForWorkspace(t, db, ownerID, "共享挂载指南")
	seedWorkspacePermission(t, db, sharedDocID, searcherID, ownerID, "viewer")
	seedWorkspaceAsset(t, db, searcherID, ownedDocID, "挂载流程.png")

	result, err := SearchWorkspace(searcherID, "挂载", 5)
	if err != nil {
		t.Fatalf("search workspace: %v", err)
	}

	if len(result.Folders) != 1 || result.Folders[0].Name != "挂载资料" {
		t.Fatalf("unexpected folder results: %+v", result.Folders)
	}
	if len(result.Documents) < 2 {
		t.Fatalf("expected owned and shared docs, got %+v", result.Documents)
	}
	if len(result.Media) != 1 || result.Media[0].Filename != "挂载流程.png" {
		t.Fatalf("unexpected media results: %+v", result.Media)
	}
	if result.Total != len(result.Folders)+len(result.Documents)+len(result.Media) {
		t.Fatalf("unexpected total: %+v", result)
	}
}

func TestSearchHandler_ReturnsSearchPayload(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	userID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, userID, "挂载文档")
	seedWorkspaceAsset(t, db, userID, docID, "挂载图.png")

	app := newWorkspaceTestApp(userID)
	req := httptest.NewRequest(http.MethodGet, "/search?q=挂载&limit=3", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Query != "挂载" {
		t.Fatalf("unexpected query: %+v", payload)
	}
	if len(payload.Documents) != 1 || payload.Documents[0].ID != docID {
		t.Fatalf("unexpected documents: %+v", payload.Documents)
	}
	if len(payload.Media) != 1 || payload.Media[0].Filename != "挂载图.png" {
		t.Fatalf("unexpected media: %+v", payload.Media)
	}
}

func TestSearchWorkspace_MatchesDocumentBodyContentJSON(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	userID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, userID, "普通标题")

	if err := db.Model(&models.DocumentBody{}).
		Where("document_id = ?", docID).
		Updates(map[string]any{
			"content_json": `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"正文里有挂载关键字"}]}]}`,
			"plain_text":   "这里故意不依赖 plain_text",
		}).Error; err != nil {
		t.Fatalf("update body content: %v", err)
	}

	result, err := SearchWorkspace(userID, "挂载关键字", 5)
	if err != nil {
		t.Fatalf("search workspace: %v", err)
	}

	if len(result.Documents) != 1 || result.Documents[0].ID != docID {
		t.Fatalf("expected body content search hit document, got %+v", result.Documents)
	}
	if !strings.Contains(result.Documents[0].Excerpt, "挂载关键字") {
		t.Fatalf("expected excerpt snippet to include keyword, got %+v", result.Documents[0].Excerpt)
	}
	if strings.HasPrefix(result.Documents[0].Excerpt, "seed") {
		t.Fatalf("expected matched snippet instead of document opening, got %+v", result.Documents[0].Excerpt)
	}
}
