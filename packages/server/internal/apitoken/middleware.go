package apitoken

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const (
	LocalsTokenID = "apiTokenId"
	LocalsScopes  = "apiTokenScopes"
)

func Protected(requiredScopes ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		rawToken, err := bearerToken(c)
		if err != nil {
			return writeAuthError(c, err)
		}

		authenticated, err := Authenticate(rawToken, c.IP())
		if err != nil {
			return writeAuthError(c, err)
		}
		if !HasScopes(authenticated.Scopes, requiredScopes...) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":          "insufficient API token scope",
				"requiredScopes": requiredScopes,
			})
		}

		c.Locals("userId", authenticated.UserID.String())
		c.Locals(LocalsTokenID, authenticated.TokenID.String())
		c.Locals(LocalsScopes, authenticated.Scopes)
		return c.Next()
	}
}

func bearerToken(c *fiber.Ctx) (string, error) {
	authHeader := strings.TrimSpace(c.Get("Authorization"))
	if authHeader == "" {
		return "", fiber.NewError(fiber.StatusUnauthorized, "missing Authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" || strings.TrimSpace(parts[1]) == "" {
		return "", fiber.NewError(fiber.StatusUnauthorized, "malformed Authorization header")
	}
	return strings.TrimSpace(parts[1]), nil
}

// authErrorCode maps an authentication failure to a stable, machine-readable
// code and human-readable description. Expired and revoked tokens are reported
// distinctly so a client knows to refresh or replace the token, while unknown or
// malformed credentials stay deliberately generic so a caller cannot probe
// whether a given token value exists.
func authErrorCode(err error) (code string, description string) {
	switch {
	case errors.Is(err, ErrTokenExpired):
		return "token_expired", ErrTokenExpired.Error()
	case errors.Is(err, ErrTokenRevoked):
		return "token_revoked", ErrTokenRevoked.Error()
	default:
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			return "invalid_token", fiberErr.Message
		}
		return "invalid_token", ErrTokenUnauthorized.Error()
	}
}

// writeAuthError renders a 401 with a machine-readable code plus an RFC 6750
// WWW-Authenticate challenge so Bearer clients can react to expiry automatically.
func writeAuthError(c *fiber.Ctx, err error) error {
	code, description := authErrorCode(err)
	c.Set(fiber.HeaderWWWAuthenticate, fmt.Sprintf("Bearer error=%q, error_description=%q", code, description))
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error": description,
		"code":  code,
	})
}
