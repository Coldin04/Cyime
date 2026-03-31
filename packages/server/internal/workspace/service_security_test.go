package workspace

import (
	"fmt"
	"testing"
	"time"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupWorkspaceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	t.Setenv("SMTP_ENABLED", "true")

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
		&models.DocumentAssetRef{},
		&models.DocumentPermission{},
		&models.DocumentInvite{},
		&models.Notification{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	database.DB = db
	return db
}

func seedVerifiedUser(t *testing.T, db *gorm.DB, userID uuid.UUID, email string) {
	t.Helper()
	var count int64
	if err := db.Model(&models.User{}).Where("id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count user: %v", err)
	}
	if count > 0 {
		return
	}
	normalizedEmail := email
	now := time.Now()
	user := models.User{
		ID:              userID,
		Email:           &normalizedEmail,
		EmailVerified:   true,
		EmailVerifiedAt: &now,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func seedDocumentForWorkspace(t *testing.T, db *gorm.DB, ownerID uuid.UUID, title string) uuid.UUID {
	t.Helper()
	seedVerifiedUser(t, db, ownerID, ownerID.String()+"@example.com")

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

func TestGetFile_Document_AllowsSharedViewer(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	viewerID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "shared-doc")
	seedWorkspacePermission(t, db, docID, viewerID, ownerID, "viewer")

	item, err := GetFile(viewerID, docID, "document")
	if err != nil {
		t.Fatalf("expected shared viewer access, got error: %v", err)
	}
	if item == nil || item.ID != docID {
		t.Fatalf("unexpected document item: %+v", item)
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

func TestBatchMoveFiles_DeniesSharedEditorForDocumentMove(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	editorID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "shared-doc")
	seedWorkspacePermission(t, db, docID, editorID, ownerID, "editor")

	resp, err := BatchMoveFiles(editorID, []ItemToMove{
		{ID: docID, Type: "document"},
	}, nil)
	if err != nil {
		t.Fatalf("batch move failed unexpectedly: %v", err)
	}
	if resp.Success {
		t.Fatal("expected batch move to report partial/failed result")
	}
	if resp.MovedCount != 0 {
		t.Fatalf("expected zero moved items, got %d", resp.MovedCount)
	}
	if len(resp.FailedItems) != 1 {
		t.Fatalf("expected one failed item, got %+v", resp.FailedItems)
	}
}

func TestBatchMoveFiles_MixedOwnedAndForeignDocuments_OnlyMovesOwned(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	otherUserID := uuid.New()

	ownerFolderID := uuid.New()
	if err := db.Create(&models.Folder{
		ID:          ownerFolderID,
		OwnerUserID: ownerID,
		Name:        "owner-folder",
		CreatedBy:   ownerID,
		UpdatedBy:   ownerID,
	}).Error; err != nil {
		t.Fatalf("create owner folder: %v", err)
	}

	otherFolderID := uuid.New()
	if err := db.Create(&models.Folder{
		ID:          otherFolderID,
		OwnerUserID: otherUserID,
		Name:        "other-folder",
		CreatedBy:   otherUserID,
		UpdatedBy:   otherUserID,
	}).Error; err != nil {
		t.Fatalf("create other folder: %v", err)
	}

	ownedDocID := seedDocumentForWorkspace(t, db, ownerID, "owned-doc")
	foreignDocID := seedDocumentForWorkspace(t, db, otherUserID, "foreign-doc")

	if err := db.Model(&models.Document{}).Where("id = ?", ownedDocID).Update("folder_id", ownerFolderID).Error; err != nil {
		t.Fatalf("attach owned doc: %v", err)
	}
	if err := db.Model(&models.Document{}).Where("id = ?", foreignDocID).Update("folder_id", otherFolderID).Error; err != nil {
		t.Fatalf("attach foreign doc: %v", err)
	}

	resp, err := BatchMoveFiles(ownerID, []ItemToMove{
		{ID: ownedDocID, Type: "document"},
		{ID: foreignDocID, Type: "document"},
	}, nil)
	if err != nil {
		t.Fatalf("batch move failed unexpectedly: %v", err)
	}
	if resp.Success {
		t.Fatal("expected partial success because one document is unauthorized")
	}
	if resp.MovedCount != 1 {
		t.Fatalf("expected one moved item, got %d", resp.MovedCount)
	}
	if len(resp.FailedItems) != 1 {
		t.Fatalf("expected one failed item, got %+v", resp.FailedItems)
	}

	var ownedDoc models.Document
	if err := db.First(&ownedDoc, "id = ?", ownedDocID).Error; err != nil {
		t.Fatalf("load owned doc: %v", err)
	}
	if ownedDoc.FolderID != nil {
		t.Fatalf("expected owned doc moved to root, got folder %v", *ownedDoc.FolderID)
	}

	var foreignDoc models.Document
	if err := db.First(&foreignDoc, "id = ?", foreignDocID).Error; err != nil {
		t.Fatalf("load foreign doc: %v", err)
	}
	if foreignDoc.FolderID == nil || *foreignDoc.FolderID != otherFolderID {
		t.Fatalf("expected foreign doc unchanged, got %+v", foreignDoc.FolderID)
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

func TestGetPublicDocument_PrivateDocumentNotExposed(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "private-doc")

	item, err := GetPublicDocument(docID, nil)
	if err == nil || item != nil {
		t.Fatalf("expected private doc to be hidden, got item=%+v err=%v", item, err)
	}
}

func TestGetPublicDocument_PublicDocumentReadable(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "public-doc")

	if err := db.Model(&models.Document{}).Where("id = ?", docID).Update("public_access", PublicAccessGlobal).Error; err != nil {
		t.Fatalf("set public_access: %v", err)
	}

	item, err := GetPublicDocument(docID, nil)
	if err != nil {
		t.Fatalf("expected public doc readable, got err=%v", err)
	}
	if item == nil || item.PublicAccess == nil || *item.PublicAccess != PublicAccessGlobal {
		t.Fatalf("expected public access in response, got item=%+v", item)
	}
}

func TestGetPublicDocument_AuthenticatedRequiresLogin(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	readerID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "auth-doc")
	seedVerifiedUser(t, db, readerID, readerID.String()+"@example.com")

	if err := db.Model(&models.Document{}).Where("id = ?", docID).Update("public_access", PublicAccessAuthenticated).Error; err != nil {
		t.Fatalf("set public_access: %v", err)
	}

	if _, err := GetPublicDocument(docID, nil); err == nil {
		t.Fatal("expected unauthenticated access to fail")
	}

	item, err := GetPublicDocument(docID, &readerID)
	if err != nil {
		t.Fatalf("expected authenticated read success, got err=%v", err)
	}
	if item == nil {
		t.Fatal("expected document item")
	}
}

func TestUpdateDocumentPublicAccess_DeniesNonOwner(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	editorID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "public-control")
	seedWorkspacePermission(t, db, docID, editorID, ownerID, "editor")

	if err := UpdateDocumentPublicAccess(editorID, docID, PublicAccessGlobal); err == nil {
		t.Fatal("expected non-owner to be denied public-access update")
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

func TestUpdateDocumentTitle_AllowsEditor(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	editorID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "shared-doc")
	seedWorkspacePermission(t, db, docID, editorID, ownerID, "editor")

	if err := UpdateDocumentTitle(editorID, docID, "updated-by-editor"); err != nil {
		t.Fatalf("expected editor title update success: %v", err)
	}

	var doc models.Document
	if err := db.First(&doc, "id = ?", docID).Error; err != nil {
		t.Fatalf("load document: %v", err)
	}
	if doc.Title != "updated-by-editor" {
		t.Fatalf("expected updated title, got %s", doc.Title)
	}
}

func TestUpdateDocumentTitle_DeniesViewer(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	viewerID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "shared-doc")
	seedWorkspacePermission(t, db, docID, viewerID, ownerID, "viewer")

	if err := UpdateDocumentTitle(viewerID, docID, "viewer-should-fail"); err == nil {
		t.Fatal("expected viewer title update to fail")
	}
}

func TestUpdateDocumentManualExcerpt_AllowsCollaborator(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	collaboratorID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "shared-doc")
	seedWorkspacePermission(t, db, docID, collaboratorID, ownerID, "collaborator")

	manualExcerpt, excerpt, err := UpdateDocumentManualExcerpt(collaboratorID, docID, "手动介绍")
	if err != nil {
		t.Fatalf("expected collaborator manual excerpt update success: %v", err)
	}
	if manualExcerpt != "手动介绍" {
		t.Fatalf("expected returned manual excerpt, got %q", manualExcerpt)
	}
	if excerpt != "手动介绍" {
		t.Fatalf("expected returned excerpt to be manual text, got %q", excerpt)
	}

	var doc models.Document
	if err := db.First(&doc, "id = ?", docID).Error; err != nil {
		t.Fatalf("load document: %v", err)
	}
	if doc.ManualExcerpt != "手动介绍" {
		t.Fatalf("expected manual excerpt saved, got %q", doc.ManualExcerpt)
	}
}

func TestUpdateDocumentManualExcerpt_DeniesEditor(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	editorID := uuid.New()
	docID := seedDocumentForWorkspace(t, db, ownerID, "shared-doc")
	seedWorkspacePermission(t, db, docID, editorID, ownerID, "editor")

	if _, _, err := UpdateDocumentManualExcerpt(editorID, docID, "editor-should-fail"); err == nil {
		t.Fatal("expected editor manual excerpt update to fail")
	}
}

func TestShareDocument_AllowsOwnerToGrantEditor(t *testing.T) {
	db := setupWorkspaceTestDB(t)
	ownerID := uuid.New()
	targetUserID := uuid.New()
	seedVerifiedUser(t, db, targetUserID, targetUserID.String()+"@example.com")
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
	seedVerifiedUser(t, db, targetUserID, targetUserID.String()+"@example.com")
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
