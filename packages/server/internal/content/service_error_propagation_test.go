package content

import (
	"errors"
	"strings"
	"testing"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/acl"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestDeleteContentByDocumentID_NoOpOnMissingDocument confirms the benign
// path still returns nil: a document that does not exist (or is owned by
// someone else) is a caller-side contract violation we deliberately swallow
// to preserve idempotency of batch delete loops.
func TestDeleteContentByDocumentID_NoOpOnMissingDocument(t *testing.T) {
	db := newContentMemoryDB(t)

	err := db.Transaction(func(tx *gorm.DB) error {
		return DeleteContentByDocumentID(tx, uuid.New(), uuid.New())
	})
	if err != nil {
		t.Fatalf("expected missing document to be a no-op, got %v", err)
	}
}

// TestDeleteContentByDocumentID_PropagatesRealDBError is the P3-#8 regression:
// before the fix, *every* error returned by the ACL lookup was silently
// dropped, so a broken database handle would report success and leave
// orphaned document_bodies rows. Shutting down the *sql.DB forces the next
// query to fail with a real "sql: database is closed" error; we assert the
// function now propagates that instead of swallowing it.
func TestDeleteContentByDocumentID_PropagatesRealDBError(t *testing.T) {
	db := newContentMemoryDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	err = DeleteContentByDocumentID(db, uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected real DB error to propagate, got nil")
	}
	if errors.Is(err, acl.ErrDocumentNotFoundOrForbidden) {
		t.Fatalf("expected infra error, got ACL sentinel: %v", err)
	}
	if !strings.Contains(err.Error(), "closed") && !strings.Contains(err.Error(), "database is closed") {
		t.Fatalf("expected a DB-closed style error, got %v", err)
	}
}

// TestRestoreContentByDocumentID_PropagatesRealDBError mirrors the delete
// test for the trash-restore code path.
func TestRestoreContentByDocumentID_PropagatesRealDBError(t *testing.T) {
	db := newContentMemoryDB(t)
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()

	err := RestoreContentByDocumentID(db, uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected real DB error to propagate, got nil")
	}
	if errors.Is(err, acl.ErrDocumentNotFoundOrForbidden) {
		t.Fatalf("expected infra error, got ACL sentinel: %v", err)
	}
}

// TestPermanentDeleteContentByDocumentID_PropagatesRealDBError is the same
// guarantee for permanent delete.
func TestPermanentDeleteContentByDocumentID_PropagatesRealDBError(t *testing.T) {
	db := newContentMemoryDB(t)
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()

	err := PermanentDeleteContentByDocumentID(db, uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected real DB error to propagate, got nil")
	}
	if errors.Is(err, acl.ErrDocumentNotFoundOrForbidden) {
		t.Fatalf("expected infra error, got ACL sentinel: %v", err)
	}
}

// newContentMemoryDB builds a minimal in-memory SQLite instance with just
// the schema the content service touches. The helper is local to this test
// file so it doesn't collide with setupContentTestDB used by the handler
// tests (which requires more tables and seeds).
func newContentMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Document{},
		&models.DocumentBody{},
		&models.DocumentPermission{},
		&models.DocumentAssetRef{},
		&models.AssetGCJob{},
		&models.Asset{},
		&models.BlobObject{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}
