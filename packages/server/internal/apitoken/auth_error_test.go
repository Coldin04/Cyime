package apitoken

import (
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"g.co1d.in/Coldin04/Cyime/server/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestAuthenticateExpiredTokenReturnsExpiredError(t *testing.T) {
	db := setupAPITokenTestDB(t)
	userID := uuid.New()
	if err := db.Create(&models.User{ID: userID}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	created, err := CreateToken(userID, CreateTokenInput{
		Name:   "expiring",
		Scopes: []string{ScopeWorkspaceRead},
	})
	if err != nil {
		t.Fatalf("CreateToken returned error: %v", err)
	}

	past := time.Now().Add(-time.Hour)
	if err := db.Model(&models.ApiToken{}).Where("id = ?", created.ID).Update("expires_at", past).Error; err != nil {
		t.Fatalf("expire token: %v", err)
	}

	if _, err := Authenticate(created.Token, "127.0.0.1"); !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("expired token error = %v, want ErrTokenExpired", err)
	}
}

func TestAuthErrorCode(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode string
	}{
		{"expired", ErrTokenExpired, "token_expired"},
		{"revoked", ErrTokenRevoked, "token_revoked"},
		{"unknown", ErrTokenUnauthorized, "invalid_token"},
		{"missing header", fiber.NewError(fiber.StatusUnauthorized, "missing Authorization header"), "invalid_token"},
	}
	for _, tc := range cases {
		code, _ := authErrorCode(tc.err)
		if code != tc.wantCode {
			t.Fatalf("%s: code = %q, want %q", tc.name, code, tc.wantCode)
		}
	}

	// Unknown/invalid tokens must stay generic so a caller cannot probe whether
	// a given token value exists.
	if _, description := authErrorCode(ErrTokenUnauthorized); description != ErrTokenUnauthorized.Error() {
		t.Fatalf("invalid token description = %q, want generic %q", description, ErrTokenUnauthorized.Error())
	}
}

func TestProtectedDistinguishesExpiredFromInvalidToken(t *testing.T) {
	db := setupAPITokenTestDB(t)
	userID := uuid.New()
	if err := db.Create(&models.User{ID: userID}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	created, err := CreateToken(userID, CreateTokenInput{
		Name:   "expiring",
		Scopes: []string{ScopeWorkspaceRead},
	})
	if err != nil {
		t.Fatalf("CreateToken returned error: %v", err)
	}
	past := time.Now().Add(-time.Hour)
	if err := db.Model(&models.ApiToken{}).Where("id = ?", created.ID).Update("expires_at", past).Error; err != nil {
		t.Fatalf("expire token: %v", err)
	}

	app := fiber.New()
	app.Get("/x", Protected(ScopeWorkspaceRead), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// Expired token: the caller should learn to refresh, not replace, the token.
	expiredCode, expiredHeader, expiredStatus := doProtectedRequest(t, app, "Bearer "+created.Token)
	if expiredStatus != fiber.StatusUnauthorized {
		t.Fatalf("expired status = %d, want 401", expiredStatus)
	}
	if expiredCode != "token_expired" {
		t.Fatalf("expired code = %q, want token_expired", expiredCode)
	}
	if !strings.Contains(expiredHeader, "token_expired") {
		t.Fatalf("expired WWW-Authenticate = %q, want to mention token_expired", expiredHeader)
	}

	// Unknown token: response must stay generic so token existence is not leaked.
	unknownCode, _, unknownStatus := doProtectedRequest(t, app, "Bearer cyime_sk_does_not_exist")
	if unknownStatus != fiber.StatusUnauthorized {
		t.Fatalf("unknown status = %d, want 401", unknownStatus)
	}
	if unknownCode != "invalid_token" {
		t.Fatalf("unknown code = %q, want invalid_token", unknownCode)
	}
}

func doProtectedRequest(t *testing.T, app *fiber.App, authorization string) (code string, wwwAuthenticate string, status int) {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, "/x", nil)
	req.Header.Set(fiber.HeaderAuthorization, authorization)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var payload struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(body, &payload)
	return payload.Code, resp.Header.Get(fiber.HeaderWWWAuthenticate), resp.StatusCode
}
