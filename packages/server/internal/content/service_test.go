package content

import (
	"fmt"
	"testing"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupContentTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Document{},
		&models.DocumentBody{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	database.DB = db
	return db
}

func seedDocumentForContent(t *testing.T, db *gorm.DB, ownerID uuid.UUID, title, contentJSON string) (uuid.UUID, uuid.UUID) {
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

	docContent := models.DocumentBody{
		ID:             uuid.New(),
		DocumentID:     doc.ID,
		ContentJSON:    contentJSON,
		PlainText:      "seed",
		ContentVersion: 1,
		UpdatedBy:      ownerID,
	}
	if err := db.Create(&docContent).Error; err != nil {
		t.Fatalf("create document content: %v", err)
	}

	return doc.ID, docContent.ID
}

func TestGetContent_DeniesCrossUserAccess(t *testing.T) {
	db := setupContentTestDB(t)
	ownerID := uuid.New()
	attackerID := uuid.New()
	docID, _ := seedDocumentForContent(t, db, ownerID, "owner-doc", `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"secret"}]}]}`)

	if _, err := GetContent(attackerID, docID); err == nil {
		t.Fatal("expected cross-user get content to fail")
	}
}

func TestUpdateContent_DeniesCrossUserAccessAndKeepsData(t *testing.T) {
	db := setupContentTestDB(t)
	ownerID := uuid.New()
	attackerID := uuid.New()
	docID, contentID := seedDocumentForContent(t, db, ownerID, "owner-doc", `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"before"}]}]}`)

	if _, err := UpdateContent(attackerID, docID, []byte(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hacked"}]}]}`)); err == nil {
		t.Fatal("expected cross-user update content to fail")
	}

	var got models.DocumentBody
	if err := db.First(&got, "id = ?", contentID).Error; err != nil {
		t.Fatalf("load content: %v", err)
	}
	expected := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"before"}]}]}`
	if got.ContentJSON != expected {
		t.Fatalf("expected content unchanged, got: %q", got.ContentJSON)
	}
	if got.UpdatedBy != ownerID {
		t.Fatalf("expected updated_by unchanged, got: %s", got.UpdatedBy)
	}
}
