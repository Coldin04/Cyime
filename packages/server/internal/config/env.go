package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// LoadDotEnv loads KEY=VALUE pairs from a dotenv-like file.
// Existing process env vars are not overridden.
func LoadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])
		if key == "" {
			continue
		}
		value = strings.Trim(value, `"'`)
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, value)
	}
	return scanner.Err()
}

func IsTrue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

// GetOptionalNonNegativeInt reads an optional non-negative integer from env.
// Empty values mean "not configured".
func GetOptionalNonNegativeInt(key string) (*int, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return nil, fmt.Errorf("%s must be a non-negative integer", key)
	}

	return &value, nil
}

// GetRealtimeWSURL returns the WebSocket URL for realtime collaboration.
// Defaults to /api/v1/realtime/ws if not configured.
func GetRealtimeWSURL() string {
	url := os.Getenv("REALTIME_WS_URL")
	if strings.TrimSpace(url) == "" {
		return "/api/v1/realtime/ws"
	}
	return strings.TrimSpace(url)
}

// GetDocumentImageMaxBytes returns the configured max upload size for document
// image uploads. Empty, invalid, or non-positive values fall back to the
// server's current default of 5 MiB so the client can mirror backend checks.
func GetDocumentImageMaxBytes() int64 {
	const fallback int64 = 5 * 1024 * 1024

	raw := strings.TrimSpace(os.Getenv("MEDIA_DOCUMENT_IMAGE_MAX_BYTES"))
	if raw == "" {
		return fallback
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}
