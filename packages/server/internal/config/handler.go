package config

import (
	"github.com/gofiber/fiber/v2"
)

// ClientConfigResponse represents the client-facing configuration
type ClientConfigResponse struct {
	RealtimeWSURL         string `json:"realtimeWsUrl"`
	DocumentImageMaxBytes int64  `json:"documentImageMaxBytes"`
}

// GetClientConfigHandler handles GET /api/v1/config
// Returns client configuration including realtime WebSocket URL
func GetClientConfigHandler(c *fiber.Ctx) error {
	response := ClientConfigResponse{
		RealtimeWSURL:         GetRealtimeWSURL(),
		DocumentImageMaxBytes: GetDocumentImageMaxBytes(),
	}
	return c.JSON(response)
}
