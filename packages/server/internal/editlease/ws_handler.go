package editlease

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"g.co1d.in/Coldin04/Cyime/server/internal/acl"
	"g.co1d.in/Coldin04/Cyime/server/internal/auth"
	"g.co1d.in/Coldin04/Cyime/server/internal/database"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	handshakeTimeout = 5 * time.Second
	pingInterval     = 30 * time.Second
)

var errAuthFrame = errors.New("invalid or missing authentication frame")

type authFrame struct {
	AccessToken string `json:"accessToken"`
}

type leaseEventFrame struct {
	Type       string `json:"type"`
	LeaseToken string `json:"leaseToken"`
}

// EventsUpgrade gates the route to genuine WebSocket upgrade requests with a
// syntactically valid document id. It runs before the protocol upgrade, so
// it can't check the caller's identity yet — browsers can't attach an
// Authorization header to a WebSocket handshake request, so that check
// happens inside EventsHandler via a first-message frame instead.
func EventsUpgrade(c *fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.NewError(fiber.StatusUpgradeRequired, "Expected a WebSocket upgrade request")
	}
	if _, err := uuid.Parse(c.Params("id")); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Document ID must be a valid UUID")
	}
	return c.Next()
}

// EventsHandler streams lease-claimed notifications for a single document.
// It authenticates with one JSON frame right after the socket opens
// ({"accessToken": "..."}), then requires read access to the document before
// subscribing. The stream never carries document content — only "the lease
// token changed" events — but who's actively editing is still document-scoped
// information, so it's gated the same way GetContentHandler is.
func EventsHandler(conn *websocket.Conn) {
	documentID, err := uuid.Parse(conn.Params("id"))
	if err != nil {
		_ = conn.Close()
		return
	}

	if err := authenticateEventsConnection(conn, documentID); err != nil {
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, err.Error()),
			time.Now().Add(time.Second),
		)
		_ = conn.Close()
		return
	}

	subID, events := Subscribe(documentID)
	defer Unsubscribe(documentID, subID)

	// Any inbound frame after the handshake is ignored — this connection only
	// ever needs to *receive*. Reading is still necessary so a client-side
	// close (or a dead TCP connection) is noticed and this goroutine exits
	// instead of leaking a subscriber forever.
	closed := make(chan struct{})
	go func() {
		defer close(closed)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case token, ok := <-events:
			if !ok {
				return
			}
			payload, marshalErr := json.Marshal(leaseEventFrame{Type: "lease_claimed", LeaseToken: token})
			if marshalErr != nil {
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ticker.C:
			// Keeps intermediate proxies from reaping an otherwise-idle
			// connection; also serves as a liveness check on this side.
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-closed:
			return
		}
	}
}

func authenticateEventsConnection(conn *websocket.Conn, documentID uuid.UUID) error {
	_ = conn.SetReadDeadline(time.Now().Add(handshakeTimeout))
	defer func() { _ = conn.SetReadDeadline(time.Time{}) }()

	_, raw, err := conn.ReadMessage()
	if err != nil {
		return errAuthFrame
	}

	var frame authFrame
	if err := json.Unmarshal(raw, &frame); err != nil || strings.TrimSpace(frame.AccessToken) == "" {
		return errAuthFrame
	}

	claims, err := parseAccessToken(frame.AccessToken)
	if err != nil {
		return errAuthFrame
	}

	if _, err := acl.CanReadDocument(database.DB, claims.UserID, documentID); err != nil {
		return errAuthFrame
	}

	return nil
}

// parseAccessToken validates an access token the same way middleware.Protected
// does. Duplicated rather than imported so this package (business logic for
// document edit leases) doesn't take on a dependency on the HTTP middleware
// package for three lines of JWT parsing.
func parseAccessToken(tokenString string) (*auth.JWTClaims, error) {
	secret, err := auth.LoadJWTSecret()
	if err != nil {
		return nil, err
	}

	claims := &auth.JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errAuthFrame
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errAuthFrame
	}
	return claims, nil
}
