package workspace

import (
	"bytes"
	"encoding/json"
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
	app.Get("/shared/documents", ListSharedDocumentsHandler)
	app.Get("/documents/:id/shares", ListDocumentMembersHandler)
	app.Post("/documents/:id/shares", ShareDocumentHandler)
	app.Post("/documents/:id/invites", InviteDocumentByEmailHandler)
	app.Delete("/documents/:id/shares/me", LeaveSharedDocumentHandler)
	app.Delete("/documents/:id/shares/:userId", RemoveDocumentMemberHandler)
	app.Get("/notifications", ListNotificationsHandler)
	app.Post("/notifications/:id/read", MarkNotificationReadHandler)
	app.Post("/document-invites/:id/accept", AcceptDocumentInviteHandler)
	app.Post("/document-invites/:id/decline", DeclineDocumentInviteHandler)
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

func TestListSharedDocumentsHandler_ReturnsSharedDocs(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	sharedUserID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "shared-doc")
	seedWorkspacePermission(t, db, docID, sharedUserID, ownerID, "viewer")

	app := newWorkspaceTestApp(sharedUserID)
	req := httptest.NewRequest(http.MethodGet, "/shared/documents", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload SharedDocumentListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Items) != 1 || payload.Items[0].DocumentID != docID {
		t.Fatalf("unexpected shared payload: %+v", payload)
	}
}

func TestShareDocumentHandler_CreatesPermission(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	targetUserID := uuid.New()
	seedVerifiedUser(t, db, targetUserID, targetUserID.String()+"@example.com")
	docID := seedDocumentForWorkspace(t, db, ownerID, "shared-doc")

	app := newWorkspaceTestApp(ownerID)
	body := bytes.NewBufferString(`{"userId":"` + targetUserID.String() + `","role":"editor"}`)
	req := httptest.NewRequest(http.MethodPost, "/documents/"+docID.String()+"/shares", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestLeaveSharedDocumentHandler_RemovesOnlySelfPermission(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	sharedUserID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "shared-doc")
	seedWorkspacePermission(t, db, docID, sharedUserID, ownerID, "editor")

	app := newWorkspaceTestApp(sharedUserID)
	req := httptest.NewRequest(http.MethodDelete, "/documents/"+docID.String()+"/shares/me", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestInviteDocumentByEmailHandler_CreatesPermissionAndNotification(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	inviteeID := uuid.New()
	seedVerifiedUser(t, db, ownerID, ownerID.String()+"@example.com")
	seedVerifiedUser(t, db, inviteeID, "invitee@example.com")
	docID := seedDocumentForWorkspace(t, db, ownerID, "shared-doc")

	app := newWorkspaceTestApp(ownerID)
	body := bytes.NewBufferString(`{"email":"invitee@example.com","role":"editor"}`)
	req := httptest.NewRequest(http.MethodPost, "/documents/"+docID.String()+"/invites", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var permission models.DocumentPermission
	if err := db.Where("document_id = ? AND user_id = ?", docID, inviteeID).First(&permission).Error; err != nil {
		t.Fatalf("load permission: %v", err)
	}
	if permission.Role != "editor" {
		t.Fatalf("expected editor role, got %s", permission.Role)
	}

	var notificationCount int64
	if err := db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", inviteeID, "document_invite").Count(&notificationCount).Error; err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if notificationCount != 1 {
		t.Fatalf("expected 1 notification, got %d", notificationCount)
	}
}

func TestDeclineDocumentInviteHandler_RemovesPermission(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	inviteeID := uuid.New()
	seedVerifiedUser(t, db, ownerID, ownerID.String()+"@example.com")
	seedVerifiedUser(t, db, inviteeID, "invitee@example.com")
	docID := seedDocumentForWorkspace(t, db, ownerID, "shared-doc")

	ownerApp := newWorkspaceTestApp(ownerID)
	inviteBody := bytes.NewBufferString(`{"email":"invitee@example.com","role":"viewer"}`)
	inviteReq := httptest.NewRequest(http.MethodPost, "/documents/"+docID.String()+"/invites", inviteBody)
	inviteReq.Header.Set("Content-Type", "application/json")
	inviteResp, err := ownerApp.Test(inviteReq, -1)
	if err != nil {
		t.Fatalf("invite request failed: %v", err)
	}
	if inviteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected invite 200, got %d", inviteResp.StatusCode)
	}

	var invite models.DocumentInvite
	if err := db.Where("document_id = ? AND invitee_user_id = ?", docID, inviteeID).First(&invite).Error; err != nil {
		t.Fatalf("load invite: %v", err)
	}

	inviteeApp := newWorkspaceTestApp(inviteeID)
	declineReq := httptest.NewRequest(http.MethodPost, "/document-invites/"+invite.ID.String()+"/decline", nil)
	declineResp, err := inviteeApp.Test(declineReq, -1)
	if err != nil {
		t.Fatalf("decline request failed: %v", err)
	}
	if declineResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected decline 204, got %d", declineResp.StatusCode)
	}

	var permissionCount int64
	if err := db.Model(&models.DocumentPermission{}).
		Where("document_id = ? AND user_id = ? AND deleted_at IS NULL", docID, inviteeID).
		Count(&permissionCount).Error; err != nil {
		t.Fatalf("count permissions: %v", err)
	}
	if permissionCount != 0 {
		t.Fatalf("expected permission removed, got %d", permissionCount)
	}
}
