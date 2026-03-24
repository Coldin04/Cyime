package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.UserSession{}, &models.UserRefreshToken{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	database.DB = db
	tokenService = nil
	return db
}

func newAuthTestApp(userID uuid.UUID) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userId", userID.String())
		return c.Next()
	})
	app.Get("/auth/sessions", HandleListSessions)
	app.Delete("/auth/sessions/others", HandleRevokeOtherSessions)
	app.Delete("/auth/sessions/:id", HandleRevokeSession)
	return app
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func seedSessionWithToken(t *testing.T, db *gorm.DB, userID uuid.UUID, userAgent string, lastSeenAt time.Time, rawRefreshToken string) models.UserSession {
	t.Helper()
	session := models.UserSession{
		ID:          uuid.New(),
		UserID:      userID,
		UserAgent:   userAgent,
		DeviceLabel: buildDeviceLabel(userAgent),
		LastSeenAt:  lastSeenAt,
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create session: %v", err)
	}
	token := models.UserRefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		SessionID: session.ID,
		TokenHash: hashToken(rawRefreshToken),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: lastSeenAt,
	}
	if err := db.Create(&token).Error; err != nil {
		t.Fatalf("create refresh token: %v", err)
	}
	return session
}

func TestHandleListSessions_ReturnsCurrentAndOtherSessions(t *testing.T) {
	db := setupAuthTestDB(t)
	userID := uuid.New()
	email := "coldin@example.com"
	if err := db.Create(&models.User{ID: userID, Email: &email}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	currentSession := seedSessionWithToken(t, db, userID, "Mozilla/5.0 Firefox", time.Now().Add(-time.Hour), "current-token")
	_ = seedSessionWithToken(t, db, userID, "Mozilla/5.0 Chrome", time.Now().Add(-2*time.Hour), "other-token")

	app := newAuthTestApp(userID)
	req := httptest.NewRequest(http.MethodGet, "/auth/sessions", nil)
	req.AddCookie(&http.Cookie{Name: "cyime_refresh_token", Value: "current-token"})
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload SessionListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Items) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(payload.Items))
	}

	var foundCurrent bool
	for _, item := range payload.Items {
		if item.ID == currentSession.ID.String() {
			foundCurrent = item.Current
		}
	}
	if !foundCurrent {
		t.Fatalf("expected current session to be marked current")
	}
}

func TestHandleRevokeOtherSessions_RevokesOnlyOtherSessions(t *testing.T) {
	db := setupAuthTestDB(t)
	userID := uuid.New()
	email := "coldin@example.com"
	if err := db.Create(&models.User{ID: userID, Email: &email}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	currentSession := seedSessionWithToken(t, db, userID, "Mozilla/5.0 Firefox", time.Now().Add(-time.Hour), "current-token")
	otherSession := seedSessionWithToken(t, db, userID, "Mozilla/5.0 Chrome", time.Now().Add(-2*time.Hour), "other-token")

	app := newAuthTestApp(userID)
	req := httptest.NewRequest(http.MethodDelete, "/auth/sessions/others", nil)
	req.AddCookie(&http.Cookie{Name: "cyime_refresh_token", Value: "current-token"})
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var currentCount int64
	if err := db.Model(&models.UserSession{}).Where("id = ? AND revoked_at IS NULL", currentSession.ID).Count(&currentCount).Error; err != nil {
		t.Fatalf("count current session: %v", err)
	}
	if currentCount != 1 {
		t.Fatalf("expected current session kept")
	}

	var revokedAt models.UserSession
	if err := db.First(&revokedAt, "id = ?", otherSession.ID).Error; err != nil {
		t.Fatalf("load other session: %v", err)
	}
	if revokedAt.RevokedAt == nil {
		t.Fatalf("expected other session revoked")
	}
}

func TestFindOrCreateUser_AllowsMultipleUsersWithoutEmail(t *testing.T) {
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.UserIdentityProvider{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	profileA := &UserProfile{
		Subject: "github-user-a",
		Email:   "",
		Name:    "User A",
	}
	profileB := &UserProfile{
		Subject: "github-user-b",
		Email:   "",
		Name:    "User B",
	}

	userA, err := findOrCreateUser(db, "github", profileA)
	if err != nil {
		t.Fatalf("create user A: %v", err)
	}
	userB, err := findOrCreateUser(db, "github", profileB)
	if err != nil {
		t.Fatalf("create user B: %v", err)
	}

	if userA.Email != nil {
		t.Fatalf("expected user A email to be nil, got %q", *userA.Email)
	}
	if userB.Email != nil {
		t.Fatalf("expected user B email to be nil, got %q", *userB.Email)
	}
	if userA.ID == userB.ID {
		t.Fatalf("expected different users to be created")
	}
}
