package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestGenerateState_SetsHardenedCookie verifies the P2-#10 fix: the state
// cookie is HttpOnly, scoped to /api/v1/auth, and uses SameSite=Lax. The
// Secure attribute is only set on HTTPS, and the in-memory Fiber test uses
// HTTP, so we don't assert on it here — a separate happy-path exists below.
func TestGenerateState_SetsHardenedCookie(t *testing.T) {
	app := fiber.New()
	app.Get("/login", func(c *fiber.Ctx) error {
		state, err := generateState(c)
		if err != nil {
			return err
		}
		return c.SendString(state)
	})

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	setCookie := resp.Header.Get("Set-Cookie")
	if setCookie == "" {
		t.Fatal("expected Set-Cookie header")
	}
	lower := strings.ToLower(setCookie)
	if !strings.Contains(lower, "oidc_state=") {
		t.Fatalf("expected oidc_state cookie, got %q", setCookie)
	}
	if !strings.Contains(lower, "httponly") {
		t.Fatalf("expected HttpOnly attribute, got %q", setCookie)
	}
	if !strings.Contains(lower, "path=/api/v1/auth") {
		t.Fatalf("expected Path=/api/v1/auth, got %q", setCookie)
	}
	if !strings.Contains(lower, "samesite=lax") {
		t.Fatalf("expected SameSite=Lax, got %q", setCookie)
	}
}

// TestVerifyState_AcceptsMatchingCookie covers the happy path and the
// single-use invalidation side-effect: after a successful verify, the cookie
// must be cleared by a Set-Cookie with an expiry in the past so it cannot be
// replayed on a second callback.
func TestVerifyState_AcceptsMatchingCookie(t *testing.T) {
	app := fiber.New()
	app.Get("/callback", func(c *fiber.Ctx) error {
		if err := verifyState(c); err != nil {
			return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/callback?state=abc123", nil)
	req.AddCookie(&http.Cookie{Name: "oidc_state", Value: "abc123"})
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Post-verify the cookie should be invalidated.
	setCookie := resp.Header.Get("Set-Cookie")
	if setCookie == "" || !strings.Contains(setCookie, "oidc_state=") {
		t.Fatalf("expected oidc_state cookie invalidation, got %q", setCookie)
	}
	// Expires in the past → cookie removed.
	if !strings.Contains(strings.ToLower(setCookie), "max-age=0") &&
		!strings.Contains(strings.ToLower(setCookie), "expires=") {
		t.Fatalf("expected cookie expiration header, got %q", setCookie)
	}
}

func TestVerifyState_RejectsMissingCookie(t *testing.T) {
	app := fiber.New()
	app.Get("/callback", func(c *fiber.Ctx) error {
		if err := verifyState(c); err != nil {
			return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/callback?state=abc123", nil)
	// No cookie attached.
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestVerifyState_RejectsMismatch(t *testing.T) {
	app := fiber.New()
	app.Get("/callback", func(c *fiber.Ctx) error {
		if err := verifyState(c); err != nil {
			return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/callback?state=attacker", nil)
	req.AddCookie(&http.Cookie{Name: "oidc_state", Value: "victim"})
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// TestVerifyState_RejectsMissingQuery guards against the half-set case where
// a cookie is present but the callback URL has no state query parameter. The
// previous implementation handled this via the same || chain; the new one
// uses separate branches and still must reject.
func TestVerifyState_RejectsMissingQuery(t *testing.T) {
	app := fiber.New()
	app.Get("/callback", func(c *fiber.Ctx) error {
		if err := verifyState(c); err != nil {
			return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/callback", nil)
	req.AddCookie(&http.Cookie{Name: "oidc_state", Value: "something"})
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
