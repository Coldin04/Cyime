package auth

import (
	"context"
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
	"golang.org/x/oauth2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// LoadJWTSecret enforces a minimum length and rejects known defaults; this
	// value is long enough and not on the blocklist so the auth handlers can
	// construct a TokenService without touching the operator-facing env.
	t.Setenv("JWT_SECRET_KEY", "test-secret-please-rotate-aaaaaaaaaaaaaaaa")
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

func TestFindOrCreateUser_MergesVerifiedEmailAcrossProviders(t *testing.T) {
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.UserIdentityProvider{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	email := "same@example.com"
	existing := models.User{
		ID:            uuid.New(),
		Email:         &email,
		EmailVerified: true,
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("create existing user: %v", err)
	}

	profile := &UserProfile{
		Subject:       "oidc-sub-1",
		Email:         "same@example.com",
		EmailVerified: true,
		Name:          "Merged User",
	}

	user, err := findOrCreateUser(db, "oidc", profile)
	if err != nil {
		t.Fatalf("find or create user: %v", err)
	}
	if user.ID != existing.ID {
		t.Fatalf("expected merge to existing user %s, got %s", existing.ID, user.ID)
	}

	var identity models.UserIdentityProvider
	if err := db.Where("provider_name = ? AND provider_user_id = ?", "oidc", "oidc-sub-1").First(&identity).Error; err != nil {
		t.Fatalf("load identity: %v", err)
	}
	if identity.UserID != existing.ID {
		t.Fatalf("expected identity to link to existing user, got %s", identity.UserID)
	}
}

func TestFindOrCreateUser_DeniesUnverifiedEmailMerge(t *testing.T) {
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.UserIdentityProvider{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	email := "same@example.com"
	existing := models.User{
		ID:            uuid.New(),
		Email:         &email,
		EmailVerified: true,
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("create existing user: %v", err)
	}

	profile := &UserProfile{
		Subject:       "oidc-sub-2",
		Email:         "same@example.com",
		EmailVerified: false,
		Name:          "No Merge",
	}

	if _, err := findOrCreateUser(db, "oidc", profile); err == nil {
		t.Fatalf("expected unverified email merge to fail")
	}
}

func TestGetUserProfile_ParsesGoogleOAuthUserInfoAsTrusted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"google-user-1","email":"person@example.com","verified_email":false,"name":"Google User","picture":"https://example.com/avatar.png"}`))
	}))
	defer server.Close()

	provider := &models.AuthProvider{
		Name:         "google",
		ProtocolType: "oauth2",
		UserInfoURL:  &server.URL,
		ClientID:     "test-client",
	}
	oauth2Config := &oauth2.Config{}
	token := &oauth2.Token{AccessToken: "token", TokenType: "Bearer"}

	profile, err := getUserProfile(context.Background(), provider, oauth2Config, token)
	if err != nil {
		t.Fatalf("get user profile: %v", err)
	}
	if profile.Subject != "google-user-1" {
		t.Fatalf("expected subject google-user-1, got %q", profile.Subject)
	}
	if profile.Email != "person@example.com" {
		t.Fatalf("expected email person@example.com, got %q", profile.Email)
	}
	if !profile.EmailVerified {
		t.Fatalf("expected google email to be trusted by default")
	}
	if profile.Name != "Google User" {
		t.Fatalf("expected name Google User, got %q", profile.Name)
	}
	if profile.Picture != "https://example.com/avatar.png" {
		t.Fatalf("expected picture propagated, got %q", profile.Picture)
	}
}
