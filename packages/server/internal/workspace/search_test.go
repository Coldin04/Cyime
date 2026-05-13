package workspace

import (
	"encoding/json"
	"fmt"
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

func TestSearchHandler_SearchesOwnedDocumentsWhenCollaborationOff(t *testing.T) {
	t.Setenv("COLLABORATION_ENABLED", "false")

	db := setupWorkspaceTestDB(t)
	userID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, userID, "挂载文档")

	app := newWorkspaceTestApp(userID)
	req := httptest.NewRequest(http.MethodGet, "/search?q=挂载&limit=3", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 when collaboration disabled, got %d", resp.StatusCode)
	}

	var payload SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Documents) != 1 || payload.Documents[0].ID != docID {
		t.Fatalf("expected owned document search results, got %+v", payload.Documents)
	}
}

func TestSearchWorkspace_MatchesDocumentBodyContentJSON(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	userID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, userID, "普通标题")

	if err := db.Model(&models.DocumentBody{}).
		Where("document_id = ?", docID).
		Updates(map[string]any{
			"content_json": `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"这是一个用于测试挂载关键字的文档"}]}]}`,
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

func TestSearchWorkspace_FuzzyFallbackDoesNotLoadContentJSON(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	userID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, userID, "普通标题")

	largeContentJSON := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"` + strings.Repeat("a", 10000) + `"}]}]}`
	if err := db.Model(&models.DocumentBody{}).
		Where("document_id = ?", docID).
		Updates(map[string]any{
			"content_json": largeContentJSON,
			"plain_text":   "alpha body zebra",
		}).Error; err != nil {
		t.Fatalf("update body content: %v", err)
	}

	rows, err := fetchAccessibleDocumentSearchRows(userID, nil, 5, false)
	if err != nil {
		t.Fatalf("fetch fuzzy candidates: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != docID {
		t.Fatalf("expected seeded document candidate, got %+v", rows)
	}
	if rows[0].ContentJSON != "" {
		t.Fatalf("expected unfiltered fuzzy candidate to omit content_json, got %d bytes", len(rows[0].ContentJSON))
	}
	if rows[0].PlainText != "alpha body zebra" {
		t.Fatalf("expected lightweight plain_text to remain available, got %q", rows[0].PlainText)
	}

	result, err := SearchWorkspace(userID, "abz", 5)
	if err != nil {
		t.Fatalf("search workspace: %v", err)
	}
	if len(result.Documents) != 1 || result.Documents[0].ID != docID {
		t.Fatalf("expected fuzzy fallback to still match plain_text, got %+v", result.Documents)
	}
}

func TestSearchWorkspace_MultiKeywordSnippetPrefersMatchedContext(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	userID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, userID, "普通标题")

	if err := db.Model(&models.DocumentBody{}).
		Where("document_id = ?", docID).
		Updates(map[string]any{
			"content_json": `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"前面一段无关内容。这里提到青柠。再往后还有很多内容，最后才提到独特的香气和口感。"}]}]}`,
			"plain_text":   "前面一段无关内容。这里提到青柠。再往后还有很多内容，最后才提到独特的香气和口感。",
		}).Error; err != nil {
		t.Fatalf("update body content: %v", err)
	}

	result, err := SearchWorkspace(userID, "独特 青柠", 5)
	if err != nil {
		t.Fatalf("search workspace: %v", err)
	}
	if len(result.Documents) != 1 {
		t.Fatalf("expected single document hit, got %+v", result.Documents)
	}
	if !strings.Contains(result.Documents[0].Excerpt, "青柠") || !strings.Contains(result.Documents[0].Excerpt, "独特") {
		t.Fatalf("expected snippet to include both keyword contexts, got %+v", result.Documents[0].Excerpt)
	}
}

func TestSearchWorkspace_MultiKeywordMediaMatchesAcrossTerms(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	userID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, userID, "青柠笔记")
	seedWorkspaceAsset(t, db, userID, docID, "独特香气.png")

	result, err := SearchWorkspace(userID, "青柠 独特", 5)
	if err != nil {
		t.Fatalf("search workspace: %v", err)
	}
	if len(result.Media) != 1 {
		t.Fatalf("expected multi-keyword media hit, got %+v", result.Media)
	}
}

func TestSearchWorkspace_LimitsLongMultiKeywordQueries(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	userID := uuid.New()
	_ = seedDocumentForWorkspace(t, db, userID, "普通标题")

	terms := make([]string, 0, 1000)
	for i := 0; i < 1000; i++ {
		terms = append(terms, fmt.Sprintf("term%04d", i))
	}
	query := strings.Join(terms, " ")

	if got := tokenizeSearchTerms(query); len(got) != maxSearchTerms {
		t.Fatalf("expected tokenized terms to be capped at %d, got %d", maxSearchTerms, len(got))
	}
	if got := len([]rune(normalizeSearchQuery(query))); got > maxSearchQueryRunes {
		t.Fatalf("expected normalized query to be capped at %d runes, got %d", maxSearchQueryRunes, got)
	}

	result, err := SearchWorkspace(userID, query, 5)
	if err != nil {
		t.Fatalf("search workspace with many terms should not build an oversized SQL expression: %v", err)
	}
	if result.Query != normalizeSearchQuery(query) {
		t.Fatalf("expected returned query to be normalized, got %q", result.Query)
	}
}

func TestSearchWorkspace_FuzzyAndRankedResults(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	userID := uuid.New()

	topDocID := seedDocumentForWorkspace(t, db, userID, "挂载指南")
	_ = seedDocumentForWorkspace(t, db, userID, "云端挂载实践记录")

	folderA := models.Folder{
		ID:          uuid.New(),
		OwnerUserID: userID,
		Name:        "挂载资料",
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}
	folderB := models.Folder{
		ID:          uuid.New(),
		OwnerUserID: userID,
		Name:        "我的挂载备忘",
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}
	if err := db.Create(&folderA).Error; err != nil {
		t.Fatalf("create folderA: %v", err)
	}
	if err := db.Create(&folderB).Error; err != nil {
		t.Fatalf("create folderB: %v", err)
	}

	seedWorkspaceAsset(t, db, userID, topDocID, "挂载说明.png")

	result, err := SearchWorkspace(userID, "挂南", 5)
	if err != nil {
		t.Fatalf("search workspace: %v", err)
	}

	if len(result.Documents) == 0 || result.Documents[0].ID != topDocID {
		t.Fatalf("expected fuzzy title match to rank first, got %+v", result.Documents)
	}

	result, err = SearchWorkspace(userID, "挂资", 5)
	if err != nil {
		t.Fatalf("search workspace folders: %v", err)
	}
	if len(result.Folders) == 0 || result.Folders[0].Name != "挂载资料" {
		t.Fatalf("expected stronger folder name match to rank first, got %+v", result.Folders)
	}

	result, err = SearchWorkspace(userID, "挂说", 5)
	if err != nil {
		t.Fatalf("search workspace media: %v", err)
	}
	if len(result.Media) == 0 || result.Media[0].Filename != "挂载说明.png" {
		t.Fatalf("expected fuzzy media filename match, got %+v", result.Media)
	}
}
