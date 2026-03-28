package workspace

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func newWorkspaceTestApp(userID uuid.UUID) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userId", userID.String())
		return c.Next()
	})
	app.Get("/files/:id", GetFileHandler)
	app.Delete("/files/:id", DeleteFileHandler)
	app.Post("/files/batch-delete", BatchDeleteHandler)
	app.Post("/files/batch-move", BatchMoveHandler)
	return app
}

func seedFolderForWorkspace(t *testing.T, db *gorm.DB, ownerID uuid.UUID, name string) uuid.UUID {
	t.Helper()

	folder := models.Folder{
		ID:          uuid.New(),
		OwnerUserID: ownerID,
		Name:        name,
		CreatedBy:   ownerID,
		UpdatedBy:   ownerID,
	}
	if err := db.Create(&folder).Error; err != nil {
		t.Fatalf("create folder: %v", err)
	}

	return folder.ID
}

func TestGetFileHandler_Document_CrossUserDenied(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	attackerID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "owner-doc")

	app := newWorkspaceTestApp(attackerID)
	req := httptest.NewRequest(http.MethodGet, "/files/"+docID.String()+"?type=document", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteFileHandler_Document_CrossUserDeniedAndNotDeleted(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	attackerID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "owner-doc")

	app := newWorkspaceTestApp(attackerID)
	req := httptest.NewRequest(http.MethodDelete, "/files/"+docID.String()+"?type=document", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var got models.Document
	if err := db.First(&got, "id = ?", docID).Error; err != nil {
		t.Fatalf("load document: %v", err)
	}
	if got.DeletedAt.Valid {
		t.Fatal("expected document to remain undeleted")
	}
}

func TestBatchDeleteHandler_Document_CrossUserDeniedAndNotDeleted(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	attackerID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "owner-doc")

	app := newWorkspaceTestApp(attackerID)
	body := bytes.NewBufferString(`{"items":[{"id":"` + docID.String() + `","type":"document"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/files/batch-delete", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d", resp.StatusCode)
	}

	var got models.Document
	if err := db.First(&got, "id = ?", docID).Error; err != nil {
		t.Fatalf("load document: %v", err)
	}
	if got.DeletedAt.Valid {
		t.Fatal("expected document to remain undeleted")
	}
}

func TestBatchMoveHandler_Document_CrossUserDeniedAndNotMoved(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	attackerID := uuid.New()
	folderID := seedFolderForWorkspace(t, db, ownerID, "owner-folder")
	docID := seedDocumentForWorkspace(t, db, ownerID, "owner-doc")
	if err := db.Model(&models.Document{}).Where("id = ?", docID).Update("folder_id", folderID).Error; err != nil {
		t.Fatalf("attach document to folder: %v", err)
	}

	app := newWorkspaceTestApp(attackerID)
	body := bytes.NewBufferString(`{"items":[{"id":"` + docID.String() + `","type":"document"}],"destinationFolderId":null}`)
	req := httptest.NewRequest(http.MethodPost, "/files/batch-move", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d", resp.StatusCode)
	}

	var got models.Document
	if err := db.First(&got, "id = ?", docID).Error; err != nil {
		t.Fatalf("load document: %v", err)
	}
	if got.FolderID == nil || *got.FolderID != folderID {
		t.Fatal("expected document folder unchanged")
	}
}
