package securevalue

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
)

const encryptedValuePrefix = "enc:v1:"

// EncryptedValuePrefix is the prefix used for encrypted values stored in the database.
const EncryptedValuePrefix = encryptedValuePrefix

// ErrInvalidFormat is returned by DecryptString when the input is not a valid
// encrypted value (e.g. plaintext stored before encryption was introduced).
var ErrInvalidFormat = errors.New("invalid encrypted value format")

func EncryptString(plaintext string) (string, error) {
	key, err := loadKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	payload := append(nonce, ciphertext...)
	return encryptedValuePrefix + base64.StdEncoding.EncodeToString(payload), nil
}

func DecryptString(encrypted string) (string, error) {
	key, err := loadKey()
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(encrypted, encryptedValuePrefix) {
		return "", ErrInvalidFormat
	}

	encoded := strings.TrimPrefix(encrypted, encryptedValuePrefix)
	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(payload) < nonceSize {
		return "", errors.New("invalid encrypted value payload")
	}

	nonce := payload[:nonceSize]
	ciphertext := payload[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func loadKey() ([]byte, error) {
	secret := strings.TrimSpace(os.Getenv("APP_ENCRYPTION_KEY"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("JWT_SECRET_KEY"))
	}
	if secret == "" {
		return nil, errors.New("missing APP_ENCRYPTION_KEY (or JWT_SECRET_KEY fallback)")
	}

	sum := sha256.Sum256([]byte(secret))
	return sum[:], nil
}
