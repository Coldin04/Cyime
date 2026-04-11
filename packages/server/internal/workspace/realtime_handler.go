package workspace

import (
	"errors"
	"log"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/acl"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/content"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
		if !errors.Is(err, acl.ErrDocumentNotFoundOrForbidden) {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error:   "Internal Server Error",
				Message: "Failed to authorize document read",
			})
		}
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
		YjsVersion:     docBody.YjsVersion,
	}

	return c.JSON(response)
}

// UpdateYjsStateHandler handles PUT /api/v1/realtime/documents/:id/state.
//
// The handler is the persistence layer for the realtime collaboration server,
// which mediates Yjs CRDT updates in memory. To prevent two classes of bugs:
//
//  1. Silent loss when the document_bodies row does not exist yet (the original
//     code blindly issued an UPDATE and ignored RowsAffected, returning 200
//     while writing nothing).
//  2. Last-writer-wins overwrite by stale or malicious clients (no merge, no
//     concurrency control).
//
// the handler now (a) creates the row when missing and (b) requires callers to
// echo the YjsVersion they last observed; mismatches yield 409 Conflict so the
// client can re-load and retry. ExpectedYjsVersion <= 0 is permitted only when
// no row exists yet (initial create).
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

	document, err := acl.CanEditDocument(database.DB, userID, documentID)
	if err != nil {
		if !errors.Is(err, acl.ErrDocumentNotFoundOrForbidden) {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error:   "Internal Server Error",
				Message: "Failed to authorize document edit",
			})
		}
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "Not Found",
			Message: err.Error(),
		})
	}

	var newVersion int64
	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.DocumentBody
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("document_id = ?", documentID).
			First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			// First save for this document. The realtime client has nothing
			// to echo back yet; create the row at version 1.
			newVersion = 1
			log.Printf("[RealtimeState] create doc=%s user=%s yjsVersion=%d hasContentJSON=%t", documentID, userID, newVersion, len(req.ContentJSON) > 0)
			_, err := content.PersistCanonicalContent(tx, document, userID, req.ContentJSON, &content.DocumentBodyPatch{
				YjsState:       &req.YjsState,
				YjsStateVector: &req.YjsStateVector,
				YjsVersion:     &newVersion,
			})
			return err

		case err != nil:
			return err
		}

		// Row exists. Apply optimistic concurrency control: refuse the write
		// unless the caller proves it last saw the current version. The check
		// lives in the WHERE clause so a concurrent writer racing the same
		// transaction cannot squeeze through.
		if req.ExpectedYjsVersion != existing.YjsVersion {
			log.Printf("[RealtimeState] conflict doc=%s user=%s expected=%d current=%d", documentID, userID, req.ExpectedYjsVersion, existing.YjsVersion)
			return &yjsVersionConflictError{current: existing.YjsVersion}
		}

		newVersion = existing.YjsVersion + 1
		if len(req.ContentJSON) > 0 {
			log.Printf("[RealtimeState] update doc=%s user=%s newYjsVersion=%d hasContentJSON=true", documentID, userID, newVersion)
			_, err := content.PersistCanonicalContent(tx, document, userID, req.ContentJSON, &content.DocumentBodyPatch{
				YjsState:       &req.YjsState,
				YjsStateVector: &req.YjsStateVector,
				YjsVersion:     &newVersion,
			})
			return err
		}
		log.Printf("[RealtimeState] update doc=%s user=%s newYjsVersion=%d hasContentJSON=false", documentID, userID, newVersion)
		return tx.Model(&models.DocumentBody{}).
			Where("document_id = ?", documentID).
			Updates(map[string]any{
				"yjs_state":        req.YjsState,
				"yjs_state_vector": req.YjsStateVector,
				"yjs_version":      newVersion,
				"updated_by":       userID,
			}).Error
	})

	if txErr != nil {
		var conflict *yjsVersionConflictError
		if errors.As(txErr, &conflict) {
			return c.Status(fiber.StatusConflict).JSON(YjsStateConflictResponse{
				Error:          "Conflict",
				Message:        "Yjs state version is stale; reload and retry",
				CurrentVersion: conflict.current,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Failed to update Yjs state",
		})
	}

	return c.JSON(GetYjsStateResponse{
		YjsState:       req.YjsState,
		YjsStateVector: req.YjsStateVector,
		YjsVersion:     newVersion,
	})
}

// yjsVersionConflictError carries the latest stored YjsVersion back to the
// outer transaction handler so it can be reported to the client.
type yjsVersionConflictError struct {
	current int64
}

func (e *yjsVersionConflictError) Error() string {
	return "yjs version conflict"
}
