package workspace

import (
	"sync"
	"time"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/acl"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	presenceTTL              = 20 * time.Second
	presenceDefaultSessionID = "default-session"
)

type documentPresenceHeartbeatRequest struct {
	SessionID string `json:"sessionId"`
}

type documentPresenceResponse struct {
	DocumentID      uuid.UUID `json:"documentId"`
	ConnectedCount  int       `json:"connectedCount"`
	UniqueUserCount int       `json:"uniqueUserCount"`
}

type presenceEntry struct {
	userID   uuid.UUID
	lastSeen time.Time
}

var (
	presenceMu    sync.Mutex
	presenceStore = map[uuid.UUID]map[string]presenceEntry{}
)

func normalizeSessionID(raw string) string {
	if raw == "" {
		return presenceDefaultSessionID
	}
	return raw
}

func cleanupPresenceLocked(now time.Time) {
	for documentID, sessions := range presenceStore {
		for sessionID, entry := range sessions {
			if now.Sub(entry.lastSeen) > presenceTTL {
				delete(sessions, sessionID)
			}
		}
		if len(sessions) == 0 {
			delete(presenceStore, documentID)
		}
	}
}

func countPresenceLocked(documentID uuid.UUID) (int, int) {
	sessions := presenceStore[documentID]
	if len(sessions) == 0 {
		return 0, 0
	}

	uniqueUsers := map[uuid.UUID]struct{}{}
	for _, entry := range sessions {
		uniqueUsers[entry.userID] = struct{}{}
	}
	return len(sessions), len(uniqueUsers)
}

func updatePresence(documentID uuid.UUID, userID uuid.UUID, sessionID string) (int, int) {
	now := time.Now()

	presenceMu.Lock()
	defer presenceMu.Unlock()

	cleanupPresenceLocked(now)

	sessions, exists := presenceStore[documentID]
	if !exists {
		sessions = map[string]presenceEntry{}
		presenceStore[documentID] = sessions
	}
	sessions[normalizeSessionID(sessionID)] = presenceEntry{
		userID:   userID,
		lastSeen: now,
	}

	return countPresenceLocked(documentID)
}

func readPresence(documentID uuid.UUID) (int, int) {
	now := time.Now()

	presenceMu.Lock()
	defer presenceMu.Unlock()

	cleanupPresenceLocked(now)
	return countPresenceLocked(documentID)
}

func parseUserAndDocumentID(c *fiber.Ctx) (uuid.UUID, uuid.UUID, error) {
	userIDStr, ok := c.Locals("userId").(string)
	if !ok {
		return uuid.Nil, uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid user context")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "User ID format is invalid")
	}

	documentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "Document ID must be a valid UUID")
	}

	return userID, documentID, nil
}

// HeartbeatDocumentPresenceHandler handles PUT /api/v1/workspace/documents/:id/presence
func HeartbeatDocumentPresenceHandler(c *fiber.Ctx) error {
	userID, documentID, err := parseUserAndDocumentID(c)
	if err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: err.Error(),
		})
	}

	if _, err := acl.CanReadDocument(database.DB, userID, documentID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "Not Found",
			Message: ErrDocumentNotFoundOrUnauthorized.Error(),
		})
	}

	var req documentPresenceHeartbeatRequest
	_ = c.BodyParser(&req)
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = c.Get("X-Presence-Session-Id", presenceDefaultSessionID)
	}

	connectedCount, uniqueUserCount := updatePresence(documentID, userID, sessionID)
	return c.JSON(documentPresenceResponse{
		DocumentID:      documentID,
		ConnectedCount:  connectedCount,
		UniqueUserCount: uniqueUserCount,
	})
}

// GetDocumentPresenceHandler handles GET /api/v1/workspace/documents/:id/presence
func GetDocumentPresenceHandler(c *fiber.Ctx) error {
	userID, documentID, err := parseUserAndDocumentID(c)
	if err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(ErrorResponse{
			Error:   "Bad Request",
			Message: err.Error(),
		})
	}

	if _, err := acl.CanReadDocument(database.DB, userID, documentID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "Not Found",
			Message: ErrDocumentNotFoundOrUnauthorized.Error(),
		})
	}

	connectedCount, uniqueUserCount := readPresence(documentID)
	return c.JSON(documentPresenceResponse{
		DocumentID:      documentID,
		ConnectedCount:  connectedCount,
		UniqueUserCount: uniqueUserCount,
	})
}
