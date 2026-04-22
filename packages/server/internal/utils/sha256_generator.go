// package utils is a utility package for the server, this is sha256_generator.go
package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// GenerateRandomSHA256 returns a hex-encoded random SHA256 hash string.
// It generates 32 random bytes and returns their SHA256 hex digest.
func GenerateRandomSHA256() string {
	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)
	sum := sha256.Sum256(randomBytes)
	return hex.EncodeToString(sum[:])
}
