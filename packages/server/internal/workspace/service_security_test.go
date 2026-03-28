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

func stringPtr(value string) *string {
	return &value
}
