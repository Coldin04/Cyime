package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// minJWTSecretLength is the minimum length we accept for JWT_SECRET_KEY. 32 bytes
// of entropy roughly matches the security margin of HS256 itself; anything shorter
// makes brute force on weak secrets feasible.
const minJWTSecretLength = 32

// jwtSecretBlocklist contains values that have appeared in this repo (or its
// .env.example) at some point and would render the system trivially exploitable
// if used in production. Reject them even when explicitly set.
var jwtSecretBlocklist = map[string]struct{}{
	"insecure-default-secret-for-dev-only": {},
	"replace-with-a-strong-secret":         {},
	"change-me":                            {},
	"changeme":                             {},
	"secret":                               {},
}

// LoadJWTSecret reads JWT_SECRET_KEY from the environment, validates it, and
// returns the bytes used to sign and verify access tokens. It is the single
// source of truth for the JWT signing key — both auth.NewTokenService and the
// middleware key function call this so that there is no way for the two halves
// of the auth system to drift apart and accept different secrets.
//
// LoadJWTSecret intentionally does not cache. Reading an env var plus a few
// string operations costs nothing compared to the HMAC verification that
// follows, and skipping the cache means tests using t.Setenv work without
// extra reset hooks.
func LoadJWTSecret() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv("JWT_SECRET_KEY"))
	if raw == "" {
		return nil, errors.New("JWT_SECRET_KEY environment variable is required and must not be empty")
	}
	// Check the blocklist before the length check so an operator who copy-pastes
	// a known-bad default sees a clear "this value is insecure" message instead
	// of a generic "too short" hint that they might fix by appending characters.
	if _, blocked := jwtSecretBlocklist[strings.ToLower(raw)]; blocked {
		return nil, errors.New("JWT_SECRET_KEY is set to a known insecure default; generate a strong random secret with `openssl rand -hex 32`")
	}
	if len(raw) < minJWTSecretLength {
		return nil, fmt.Errorf("JWT_SECRET_KEY must be at least %d characters long (got %d); generate one with `openssl rand -hex 32`", minJWTSecretLength, len(raw))
	}
	return []byte(raw), nil
}
