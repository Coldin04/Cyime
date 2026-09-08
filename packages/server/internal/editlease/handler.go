package editlease

import (
	"errors"
	"time"

	"g.co1d.in/Coldin04/Cyime/server/internal/acl"
	"g.co1d.in/Coldin04/Cyime/server/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type claimRequest struct {
	DeviceID   string `json:"deviceId"`
	LeaseToken string `json:"leaseToken"`
	TakeOver   bool   `json:"takeOver"`
}

type errorResponse struct {
	Error     string     `json:"error"`
	Message   string     `json:"message"`
	Code      string     `json:"code"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// RequireMutation rejects document mutations that do not carry the active
// edit lease. Content saves perform the same check inside their transaction.
func RequireMutation() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, documentID, ok := parseRequestIdentity(c)
		if !ok {
			return nil
		}
		if _, err := New(database.DB).Authorize(userID, documentID, c.Get(HeaderName)); err != nil {
			return writeManagerError(c, err)
		}
		return c.Next()
	}
}

func ClaimHandler(c *fiber.Ctx) error {
	userID, documentID, ok := parseRequestIdentity(c)
	if !ok {
		return nil
	}
	var request claimRequest
	if err := c.BodyParser(&request); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Bad Request", "Invalid request body", "INVALID_REQUEST", nil)
	}

	grant, err := New(database.DB).Claim(userID, documentID, request.DeviceID, request.LeaseToken, request.TakeOver)
	if err != nil {
		return writeManagerError(c, err)
	}
	return c.JSON(grant)
}

func RenewHandler(c *fiber.Ctx) error {
	userID, documentID, ok := parseRequestIdentity(c)
	if !ok {
		return nil
	}
	grant, err := New(database.DB).Renew(userID, documentID, c.Get(HeaderName))
	if err != nil {
		return writeManagerError(c, err)
	}
	return c.JSON(grant)
}

func ReleaseHandler(c *fiber.Ctx) error {
	userID, documentID, ok := parseRequestIdentity(c)
	if !ok {
		return nil
	}
	if err := New(database.DB).Release(userID, documentID, c.Get(HeaderName)); err != nil {
		return writeError(c, fiber.StatusInternalServerError, "Internal Server Error", "Failed to release edit lease", "EDIT_LEASE_RELEASE_FAILED", nil)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func parseRequestIdentity(c *fiber.Ctx) (uuid.UUID, uuid.UUID, bool) {
	userIDRaw, ok := c.Locals("userId").(string)
	if !ok {
		_ = writeError(c, fiber.StatusUnauthorized, "Unauthorized", "Invalid user context", "UNAUTHORIZED", nil)
		return uuid.Nil, uuid.Nil, false
	}
	userID, err := uuid.Parse(userIDRaw)
	if err != nil {
		_ = writeError(c, fiber.StatusBadRequest, "Invalid User ID", "User ID format is invalid", "INVALID_USER_ID", nil)
		return uuid.Nil, uuid.Nil, false
	}
	documentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		_ = writeError(c, fiber.StatusBadRequest, "Invalid Document ID", "Document ID must be a valid UUID", "INVALID_DOCUMENT_ID", nil)
		return uuid.Nil, uuid.Nil, false
	}
	return userID, documentID, true
}

func writeManagerError(c *fiber.Ctx, err error) error {
	var held *LeaseHeldError
	switch {
	case errors.As(err, &held):
		return writeError(c, fiber.StatusLocked, "Edit Locked", err.Error(), "EDIT_LEASE_HELD", &held.ExpiresAt)
	case errors.Is(err, ErrLeaseInvalid):
		return writeError(c, fiber.StatusConflict, "Edit Lease Invalid", err.Error(), "EDIT_LEASE_INVALID", nil)
	case errors.Is(err, ErrDeviceID):
		return writeError(c, fiber.StatusBadRequest, "Bad Request", err.Error(), "INVALID_DEVICE_ID", nil)
	case errors.Is(err, acl.ErrDocumentNotFoundOrForbidden):
		return writeError(c, fiber.StatusNotFound, "Not Found", err.Error(), "DOCUMENT_NOT_FOUND", nil)
	default:
		return writeError(c, fiber.StatusInternalServerError, "Internal Server Error", "Failed to manage edit lease", "EDIT_LEASE_FAILED", nil)
	}
}

func writeError(c *fiber.Ctx, status int, title, message, code string, expiresAt *time.Time) error {
	return c.Status(status).JSON(errorResponse{
		Error: title, Message: message, Code: code, ExpiresAt: expiresAt,
	})
}
