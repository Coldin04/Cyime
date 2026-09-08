package editlease

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"g.co1d.in/Coldin04/Cyime/server/internal/acl"
	"g.co1d.in/Coldin04/Cyime/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupManagerTest(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_txlock=immediate", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Document{}, &models.DocumentEditLease{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	ownerID := uuid.New()
	documentID := uuid.New()
	if err := db.Create(&models.Document{
		ID: documentID, OwnerUserID: ownerID, Title: "lease test",
		CreatedBy: ownerID, UpdatedBy: ownerID,
	}).Error; err != nil {
		t.Fatalf("seed document: %v", err)
	}
	return db, ownerID, documentID
}

func TestManagerAllowsOnlyOneDeviceAndSupportsTakeover(t *testing.T) {
	db, ownerID, documentID := setupManagerTest(t)
	manager := New(db)

	first, err := manager.Claim(ownerID, documentID, "device-one", "", false)
	if err != nil {
		t.Fatalf("claim first lease: %v", err)
	}
	if _, err := manager.Authorize(ownerID, documentID, first.Token); err != nil {
		t.Fatalf("authorize first lease: %v", err)
	}

	_, err = manager.Claim(ownerID, documentID, "device-two", "", false)
	var held *LeaseHeldError
	if !errors.As(err, &held) || !held.ExpiresAt.After(time.Now()) {
		t.Fatalf("second device should be locked, got %v", err)
	}

	second, err := manager.Claim(ownerID, documentID, "device-two", "", true)
	if err != nil {
		t.Fatalf("take over lease: %v", err)
	}
	if _, err := manager.Authorize(ownerID, documentID, first.Token); !errors.Is(err, ErrLeaseInvalid) {
		t.Fatalf("old token should be invalid after takeover, got %v", err)
	}
	if _, err := manager.Renew(ownerID, documentID, first.Token); !errors.Is(err, ErrLeaseInvalid) {
		t.Fatalf("old token should not renew after takeover, got %v", err)
	}
	if _, err := manager.Authorize(ownerID, documentID, second.Token); err != nil {
		t.Fatalf("authorize takeover lease: %v", err)
	}
}

func TestManagerReusesPresentedTokenAcrossTabsOnSameDevice(t *testing.T) {
	db, ownerID, documentID := setupManagerTest(t)
	manager := New(db)

	first, err := manager.Claim(ownerID, documentID, "device-one", "device-one-token", false)
	if err != nil {
		t.Fatalf("claim first tab: %v", err)
	}
	if first.Token != "device-one-token" {
		t.Fatalf("first claim token = %q", first.Token)
	}
	second, err := manager.Claim(ownerID, documentID, "device-one", first.Token, false)
	if err != nil {
		t.Fatalf("claim second tab: %v", err)
	}
	if second.Token != first.Token {
		t.Fatal("same device should reuse its presented token")
	}
	if _, err := manager.Claim(ownerID, documentID, "device-one", "wrong-token", false); !errors.Is(err, ErrLeaseInvalid) {
		t.Fatalf("same device with wrong token should be rejected, got %v", err)
	}
	if _, err := manager.Authorize(ownerID, documentID, first.Token); err != nil {
		t.Fatalf("first tab token should remain valid: %v", err)
	}
}

func TestManagerRenewsReleasesAndExpiresLease(t *testing.T) {
	db, ownerID, documentID := setupManagerTest(t)
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	manager := &Manager{db: db, ttl: time.Minute, now: func() time.Time { return now }}

	grant, err := manager.Claim(ownerID, documentID, "device-one", "", false)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	now = now.Add(30 * time.Second)
	renewed, err := manager.Renew(ownerID, documentID, grant.Token)
	if err != nil {
		t.Fatalf("renew: %v", err)
	}
	if !renewed.ExpiresAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("renewed expiry = %v", renewed.ExpiresAt)
	}
	if err := manager.Release(ownerID, documentID, grant.Token); err != nil {
		t.Fatalf("release: %v", err)
	}
	if _, err := manager.Authorize(ownerID, documentID, grant.Token); !errors.Is(err, ErrLeaseInvalid) {
		t.Fatalf("released lease should be invalid, got %v", err)
	}

	grant, err = manager.Claim(ownerID, documentID, "device-one", "", false)
	if err != nil {
		t.Fatalf("claim after release: %v", err)
	}
	now = now.Add(61 * time.Second)
	if _, err := manager.Authorize(ownerID, documentID, grant.Token); !errors.Is(err, ErrLeaseInvalid) {
		t.Fatalf("expired lease should be invalid, got %v", err)
	}
	if _, err := manager.Claim(ownerID, documentID, "device-two", "", false); err != nil {
		t.Fatalf("second device should claim expired lease: %v", err)
	}
}

func TestManagerRenewsExpiredLeaseWhenItWasNotReplaced(t *testing.T) {
	db, ownerID, documentID := setupManagerTest(t)
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	manager := &Manager{db: db, ttl: time.Minute, now: func() time.Time { return now }}

	grant, err := manager.Claim(ownerID, documentID, "device-one", "", false)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	// Browser timers can be throttled while a tab is in the background. If no
	// other device replaced this token, a late heartbeat must restore the lease.
	now = now.Add(2 * time.Minute)
	renewed, err := manager.Renew(ownerID, documentID, grant.Token)
	if err != nil {
		t.Fatalf("renew unchanged expired lease: %v", err)
	}
	if !renewed.ExpiresAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("renewed expiry = %v", renewed.ExpiresAt)
	}
	if _, err := manager.Authorize(ownerID, documentID, grant.Token); err != nil {
		t.Fatalf("authorize restored lease: %v", err)
	}
}

func TestManagerRejectsNonOwner(t *testing.T) {
	db, _, documentID := setupManagerTest(t)
	_, err := New(db).Claim(uuid.New(), documentID, "device-two", "", false)
	if !errors.Is(err, acl.ErrDocumentNotFoundOrForbidden) {
		t.Fatalf("non-owner claim error = %v", err)
	}
}
