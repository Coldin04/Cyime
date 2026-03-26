package securevalue

import "testing"

func TestEncryptDecryptString_RoundTrip(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "test-app-encryption-key")

	encrypted, err := EncryptString("secret-value")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if encrypted == "secret-value" {
		t.Fatalf("expected encrypted value to differ from plaintext")
	}

	decrypted, err := DecryptString(encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if decrypted != "secret-value" {
		t.Fatalf("unexpected decrypted value: %q", decrypted)
	}
}

func TestDecryptString_RejectsInvalidFormat(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "test-app-encryption-key")

	if _, err := DecryptString("plain-text"); err == nil {
		t.Fatalf("expected invalid format error")
	}
}
