package workspace

import (
	"errors"

	"g.co1d.in/Coldin04/Cyime/server/internal/acl"
	"g.co1d.in/Coldin04/Cyime/server/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetDocumentACLHandler handles GET /api/v1/workspace/documents/:id/acl.
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

	documentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid Document ID",
			Message: "Document ID must be a valid UUID",
		})
	}

	_, role, err := acl.ResolveDocumentRole(database.DB, userID, documentID)
	if err != nil {
		if !errors.Is(err, acl.ErrDocumentNotFoundOrForbidden) {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error:   "Internal Server Error",
				Message: "Failed to resolve document permissions",
			})
		}
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "Not Found",
			Message: err.Error(),
		})
	}

	return c.JSON(DocumentACLResponse{
		MyRole:           role,
		CanRead:          acl.RoleAllowsAction(role, acl.ActionRead),
		CanEdit:          acl.RoleAllowsAction(role, acl.ActionEdit),
		CanManageMembers: acl.RoleAllowsAction(role, acl.ActionManageMembers),
	})
}
