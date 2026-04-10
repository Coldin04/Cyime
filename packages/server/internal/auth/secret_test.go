package auth

import (
	"strings"
	"testing"
)

func TestLoadJWTSecret_RejectsMissing(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", "")

	_, err := LoadJWTSecret()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET_KEY is unset")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected required-secret error, got %v", err)
	}
}

func TestLoadJWTSecret_RejectsWhitespaceOnly(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", "   \t  ")

	if _, err := LoadJWTSecret(); err == nil {
		t.Fatal("expected error when JWT_SECRET_KEY is whitespace only")
	}
}

func TestLoadJWTSecret_RejectsTooShort(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", "short-secret")

	_, err := LoadJWTSecret()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET_KEY is too short")
	}
	if !strings.Contains(err.Error(), "at least") {
		t.Fatalf("expected length error, got %v", err)
	}
}

func TestLoadJWTSecret_RejectsKnownDefaults(t *testing.T) {
	cases := []string{
		"insecure-default-secret-for-dev-only",
		"replace-with-a-strong-secret",
		"change-me",
		"changeme",
		"INSECURE-DEFAULT-SECRET-FOR-DEV-ONLY", // case-insensitive
	}
	for _, value := range cases {
		t.Run(value, func(t *testing.T) {
			t.Setenv("JWT_SECRET_KEY", value)
			_, err := LoadJWTSecret()
			if err == nil {
				t.Fatalf("expected blocked default error for %q", value)
			}
			if !strings.Contains(err.Error(), "insecure default") {
				t.Fatalf("expected blocklist error, got %v", err)
			}
		})
	}
}

func TestLoadJWTSecret_AcceptsStrongSecret(t *testing.T) {
	const strong = "f3a4d6e7c1b2a8d9e0f1a2b3c4d5e6f70a1b2c3d"
	t.Setenv("JWT_SECRET_KEY", strong)

	secret, err := LoadJWTSecret()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(secret) != strong {
		t.Fatalf("expected secret to round-trip, got %q", string(secret))
	}
}

func TestLoadJWTSecret_TrimsSurroundingWhitespace(t *testing.T) {
	const strong = "f3a4d6e7c1b2a8d9e0f1a2b3c4d5e6f70a1b2c3d"
	t.Setenv("JWT_SECRET_KEY", "  "+strong+"\n")

	secret, err := LoadJWTSecret()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(secret) != strong {
		t.Fatalf("expected trimmed secret, got %q", string(secret))
	}
}
