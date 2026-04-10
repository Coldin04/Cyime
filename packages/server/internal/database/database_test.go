package database

import (
	"runtime"
	"testing"
	"time"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
)

// TestConnect_AppliesSafeSQLitePragmas runs the production Connect path against
// a temp HOME and verifies that the DSN actually delivered the safety options
// we requested. mattn/go-sqlite3 silently drops unsupported pragma flags, so
// the assertions below are the only thing that proves the fix is real.
func TestConnect_AppliesSafeSQLitePragmas(t *testing.T) {
	if runtime.GOOS == "windows" {
		// os.UserHomeDir on Windows reads USERPROFILE; this test only fakes HOME.
		t.Skip("HOME redirection skipped on windows")
	}
	t.Setenv("HOME", t.TempDir())

	// Connect calls log.Fatalf on failure, which would terminate the test
	// process. This test therefore only exercises the success path and then
	// proves the resulting DB handle actually has the requested pragmas.
	Connect()
	t.Cleanup(func() {
		if DB != nil {
			if sqlDB, err := DB.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
	})

	if DB == nil {
		t.Fatal("DB was not initialised")
	}

	var fk int
	if err := DB.Raw("PRAGMA foreign_keys").Scan(&fk).Error; err != nil {
		t.Fatalf("read foreign_keys pragma: %v", err)
	}
	if fk != 1 {
		t.Fatalf("foreign_keys = %d, want 1", fk)
	}

	var journal string
	if err := DB.Raw("PRAGMA journal_mode").Scan(&journal).Error; err != nil {
		t.Fatalf("read journal_mode pragma: %v", err)
	}
	if journal != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journal)
	}

	var busy int
	if err := DB.Raw("PRAGMA busy_timeout").Scan(&busy).Error; err != nil {
		t.Fatalf("read busy_timeout pragma: %v", err)
	}
	if busy < 5000 {
		t.Fatalf("busy_timeout = %d, want >= 5000", busy)
	}
}

// TestConnect_CascadeDeletesUserSessions verifies that the foreign-key fix
// actually causes the OnDelete:CASCADE declared on UserSession to fire. This
// is the regression test for the silent-cascade-loss part of bug P0-#2.
func TestConnect_CascadeDeletesUserSessions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME redirection skipped on windows")
	}
	t.Setenv("HOME", t.TempDir())

	Connect()
	t.Cleanup(func() {
		if DB != nil {
			if sqlDB, err := DB.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
	})

	email := "cascade@example.com"
	user := models.User{
		ID:    uuid.New(),
		Email: &email,
	}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	session := models.UserSession{
		ID:         uuid.New(),
		UserID:     user.ID,
		LastSeenAt: time.Now(),
	}
	if err := DB.Create(&session).Error; err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := DB.Unscoped().Delete(&user).Error; err != nil {
		t.Fatalf("delete user: %v", err)
	}

	var remaining int64
	if err := DB.Unscoped().Model(&models.UserSession{}).Where("id = ?", session.ID).Count(&remaining).Error; err != nil {
		t.Fatalf("count remaining sessions: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected user session to cascade-delete, %d rows still present", remaining)
	}
}
