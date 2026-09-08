package editlease

import (
	"testing"
	"time"

	"g.co1d.in/Coldin04/Cyime/server/internal/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const wsHandlerTestSecret = "ws-handler-test-secret-aaaaaaaaaaaaaaaa"

func signTestAccessToken(t *testing.T, secret string, userID uuid.UUID, expiresAt time.Time) string {
	t.Helper()
	claims := auth.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		UserID: userID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	return signed
}

func TestParseAccessTokenAcceptsValidToken(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", wsHandlerTestSecret)
	userID := uuid.New()
	token := signTestAccessToken(t, wsHandlerTestSecret, userID, time.Now().Add(time.Minute))

	claims, err := parseAccessToken(token)
	if err != nil {
		t.Fatalf("parseAccessToken: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("claims.UserID = %v, want %v", claims.UserID, userID)
	}
}

func TestParseAccessTokenRejectsExpiredToken(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", wsHandlerTestSecret)
	token := signTestAccessToken(t, wsHandlerTestSecret, uuid.New(), time.Now().Add(-time.Minute))

	if _, err := parseAccessToken(token); err == nil {
		t.Fatal("expected an error for an expired token")
	}
}

func TestParseAccessTokenRejectsTokenSignedWithAnotherSecret(t *testing.T) {
	otherSecret := "a-completely-different-secret-bbbbbbbbbbbbbbbb"
	token := signTestAccessToken(t, otherSecret, uuid.New(), time.Now().Add(time.Minute))

	t.Setenv("JWT_SECRET_KEY", wsHandlerTestSecret)
	if _, err := parseAccessToken(token); err == nil {
		t.Fatal("expected an error for a token signed with a different secret")
	}
}

func TestParseAccessTokenRejectsGarbage(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", wsHandlerTestSecret)

	if _, err := parseAccessToken("not-a-jwt"); err == nil {
		t.Fatal("expected an error for a non-JWT string")
	}
}
