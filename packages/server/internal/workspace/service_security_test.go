package workspace

import (
	"fmt"
	"testing"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupWorkspaceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.UserImageBedConfig{},
		&models.Folder{},
		&models.Document{},
		&models.DocumentBody{},
		&models.DocumentPermission{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	database.DB = db
	return db
}

func seedDocumentForWorkspace(t *testing.T, db *gorm.DB, ownerID uuid.UUID, title string) uuid.UUID {
	t.Helper()

	doc := models.Document{
		ID:           uuid.New(),
		OwnerUserID:  ownerID,
		Title:        title,
		Excerpt:      "seed",
		DocumentType: "rich_text",
		EditorType:   "tiptap",
		CreatedBy:    ownerID,
		UpdatedBy:    ownerID,
	}
	if err := db.Create(&doc).Error; err != nil {
		t.Fatalf("create document: %v", err)
	}

	content := models.DocumentBody{
		ID:             uuid.New(),
		DocumentID:     doc.ID,
		ContentJSON:    `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"seed"}]}]}`,
		PlainText:      "seed",
		ContentVersion: 1,
		UpdatedBy:      ownerID,
	}
	if err := db.Create(&content).Error; err != nil {
		t.Fatalf("create document content: %v", err)
	}

	return doc.ID
}

func seedWorkspacePermission(t *testing.T, db *gorm.DB, documentID, userID, createdBy uuid.UUID, role string) {
	t.Helper()
	permission := models.DocumentPermission{
		ID:         uuid.New(),
		DocumentID: documentID,
		UserID:     userID,
		Role:       role,
		CreatedBy:  createdBy,
	}
	if err := db.Create(&permission).Error; err != nil {
		t.Fatalf("create document permission: %v", err)
	}
}

func TestGetFile_Document_DeniesCrossUserAccess(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	attackerID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "owner-doc")

	if _, err := GetFile(attackerID, docID, "document"); err == nil {
		t.Fatal("expected cross-user file access to fail")
	}
}

func TestMoveDocument_DeniesCrossUserAccess(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	attackerID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "owner-doc")

	if _, err := MoveDocument(attackerID, docID, nil); err == nil {
		t.Fatal("expected cross-user move to fail")
	}
}

func TestDeleteFile_Document_DeniesCrossUserAccessAndKeepsRow(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	attackerID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "owner-doc")

	if err := DeleteFile(attackerID, docID, "document"); err == nil {
		t.Fatal("expected cross-user delete to fail")
	}

	var got models.Document
	if err := db.First(&got, "id = ?", docID).Error; err != nil {
		t.Fatalf("load document: %v", err)
	}
	if got.DeletedAt.Valid {
		t.Fatal("expected document to remain undeleted")
	}
}

func TestUpdateDocumentImageTarget_DeniesCrossUserAccess(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	attackerID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "owner-doc")
	config := models.UserImageBedConfig{
		ID:           uuid.New(),
		UserID:       attackerID,
		Name:         "attacker bed",
		ProviderType: "see",
		APIToken:     stringPtr("token"),
		IsEnabled:    true,
	}
	if err := db.Create(&config).Error; err != nil {
		t.Fatalf("create image bed config: %v", err)
	}

	if err := UpdateDocumentImageTarget(attackerID, docID, config.ID.String()); err == nil {
		t.Fatal("expected cross-user image target update to fail")
	}
}

func TestShareDocument_AllowsOwnerToGrantEditor(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	targetUserID := uuid.New()
	if err := db.Create(&models.User{ID: targetUserID}).Error; err != nil {
		t.Fatalf("create target user: %v", err)
	}
	docID := seedDocumentForWorkspace(t, db, ownerID, "shared-doc")

	result, err := ShareDocument(ownerID, docID, targetUserID, "editor")
	if err != nil {
		t.Fatalf("share document: %v", err)
	}
	if len(result.Members) != 2 {
		t.Fatalf("expected owner + one member, got %+v", result.Members)
	}
}

func TestShareDocument_RevivesSoftDeletedPermission(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	targetUserID := uuid.New()
	if err := db.Create(&models.User{ID: targetUserID}).Error; err != nil {
		t.Fatalf("create target user: %v", err)
	}
	docID := seedDocumentForWorkspace(t, db, ownerID, "shared-doc")

	if _, err := ShareDocument(ownerID, docID, targetUserID, "viewer"); err != nil {
		t.Fatalf("share document: %v", err)
	}
	if _, err := RemoveDocumentMember(ownerID, docID, targetUserID); err != nil {
		t.Fatalf("remove member: %v", err)
	}
	if _, err := ShareDocument(ownerID, docID, targetUserID, "editor"); err != nil {
		t.Fatalf("re-share document: %v", err)
	}

	var permission models.DocumentPermission
	if err := db.Unscoped().First(&permission, "document_id = ? AND user_id = ?", docID, targetUserID).Error; err != nil {
		t.Fatalf("load permission: %v", err)
	}
	if permission.DeletedAt.Valid {
		t.Fatalf("expected permission revived")
	}
	if permission.Role != "editor" {
		t.Fatalf("expected role updated, got %s", permission.Role)
	}
}

func TestListSharedDocuments_ReturnsPermissionedDocs(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	sharedUserID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "shared-doc")
	seedWorkspacePermission(t, db, docID, sharedUserID, ownerID, "viewer")

	result, err := ListSharedDocuments(sharedUserID, 20, 0)
	if err != nil {
		t.Fatalf("list shared documents: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].DocumentID != docID {
		t.Fatalf("unexpected shared documents: %+v", result.Items)
	}
	if result.Items[0].MyRole != "viewer" {
		t.Fatalf("expected viewer role, got %+v", result.Items[0])
	}
}

func TestLeaveSharedDocument_RemovesPermissionOnly(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	sharedUserID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "shared-doc")
	seedWorkspacePermission(t, db, docID, sharedUserID, ownerID, "editor")

	if err := LeaveSharedDocument(sharedUserID, docID); err != nil {
		t.Fatalf("leave shared document: %v", err)
	}

	var permissionCount int64
	if err := db.Model(&models.DocumentPermission{}).Where("document_id = ? AND user_id = ?", docID, sharedUserID).Count(&permissionCount).Error; err != nil {
		t.Fatalf("count permissions: %v", err)
	}
	if permissionCount != 0 {
		t.Fatalf("expected permission removed, got %d", permissionCount)
	}

	var doc models.Document
	if err := db.First(&doc, "id = ?", docID).Error; err != nil {
		t.Fatalf("load document: %v", err)
	}
	if doc.DeletedAt.Valid {
		t.Fatalf("expected document untouched")
	}
}

func stringPtr(value string) *string {
	return &value
}
