package config

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestGetClientConfigHandlerOmitsRealtimeConfiguration(t *testing.T) {
	t.Setenv("COLLABORATION_ENABLED", "false")
	t.Setenv("MEDIA_DOCUMENT_IMAGE_MAX_BYTES", "1234")

	app := fiber.New()
	app.Get("/config", GetClientConfigHandler)

	response, err := app.Test(httptest.NewRequest("GET", "/config", nil), -1)
	if err != nil {
		t.Fatalf("request config: %v", err)
	}
	defer response.Body.Close()

	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode config response: %v", err)
	}

	if _, exists := payload["realtimeWsUrl"]; exists {
		t.Fatal("config response must not expose realtimeWsUrl")
	}
	if enabled, ok := payload["collaborationEnabled"].(bool); !ok || enabled {
		t.Fatalf("collaborationEnabled = %#v, want false", payload["collaborationEnabled"])
	}
	if maxBytes, ok := payload["documentImageMaxBytes"].(float64); !ok || maxBytes != 1234 {
		t.Fatalf("documentImageMaxBytes = %#v, want 1234", payload["documentImageMaxBytes"])
	}
}
