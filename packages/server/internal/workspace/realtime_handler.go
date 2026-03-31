package workspace

import (
	"errors"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/acl"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetDocumentACLHandler handles GET /api/v1/workspace/documents/:id/acl
func GetDocumentACLHandler(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "Unauthorized",
			Message: "Invalid user context",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid User ID",
			Message: "User ID format is invalid",
		})
	}

	documentIDStr := c.Params("id")
	documentID, err := uuid.Parse(documentIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid Document ID",
			Message: "Document ID must be a valid UUID",
		})
	}

	_, role, err := acl.ResolveDocumentRole(database.DB, userID, documentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "Not Found",
			Message: err.Error(),
		})
	}

	// Determine capabilities based on role
	canRead := acl.RoleAllowsAction(role, acl.ActionRead)
	canEdit := acl.RoleAllowsAction(role, acl.ActionEdit)
	canManageMembers := acl.RoleAllowsAction(role, acl.ActionManageMembers)

	response := DocumentACLResponse{
		MyRole:           role,
		CanRead:          canRead,
		CanEdit:          canEdit,
		CanManageMembers: canManageMembers,
	}

	return c.JSON(response)
}

// GetYjsStateHandler handles GET /api/v1/realtime/documents/:id/state
func GetYjsStateHandler(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "Unauthorized",
			Message: "Invalid user context",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid User ID",
			Message: "User ID format is invalid",
		})
	}

	documentIDStr := c.Params("id")
	documentID, err := uuid.Parse(documentIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid Document ID",
			Message: "Document ID must be a valid UUID",
		})
	}

	_, err = acl.CanReadDocument(database.DB, userID, documentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "Not Found",
			Message: err.Error(),
		})
	}

	var docBody models.DocumentBody
	if err := database.DB.Where("document_id = ?", documentID).First(&docBody).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error:   "Not Found",
				Message: "Document content not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Failed to retrieve document state",
		})
	}

	response := GetYjsStateResponse{
		YjsState:       docBody.YjsState,
		YjsStateVector: docBody.YjsStateVector,
	}

	return c.JSON(response)
}

// UpdateYjsStateHandler handles PUT /api/v1/realtime/documents/:id/state
func UpdateYjsStateHandler(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "Unauthorized",
			Message: "Invalid user context",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid User ID",
			Message: "User ID format is invalid",
		})
	}

	documentIDStr := c.Params("id")
	documentID, err := uuid.Parse(documentIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid Document ID",
			Message: "Document ID must be a valid UUID",
		})
	}

	var req UpdateYjsStateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid request body",
		})
	}

	_, err = acl.CanEditDocument(database.DB, userID, documentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "Not Found",
			Message: err.Error(),
		})
	}

	// Update document body with new Yjs state
	if err := database.DB.Model(&models.DocumentBody{}).
		Where("document_id = ?", documentID).
		Updates(map[string]interface{}{
			"yjs_state":        req.YjsState,
			"yjs_state_vector": req.YjsStateVector,
			"updated_by":       userID,
		}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Failed to update Yjs state",
		})
	}

	response := GetYjsStateResponse{
		YjsState:       req.YjsState,
		YjsStateVector: req.YjsStateVector,
	}

	return c.JSON(response)
}