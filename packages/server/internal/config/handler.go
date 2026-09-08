package config

import (
	"github.com/gofiber/fiber/v2"
)

// ClientConfigResponse represents the client-facing configuration
type ClientConfigResponse struct {
	CollaborationEnabled  bool  `json:"collaborationEnabled"`
	DocumentImageMaxBytes int64 `json:"documentImageMaxBytes"`
}

// GetClientConfigHandler handles GET /api/v1/config
func GetClientConfigHandler(c *fiber.Ctx) error {
	response := ClientConfigResponse{
		CollaborationEnabled:  GetCollaborationEnabled(),
		DocumentImageMaxBytes: GetDocumentImageMaxBytes(),
	}
	return c.JSON(response)
}
