package editlease

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"g.co1d.in/Coldin04/Cyime/server/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func newLeaseTestApp(userID uuid.UUID) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userId", userID.String())
		return c.Next()
	})
	app.Post("/documents/:id/lease", ClaimHandler)
	return app
}

func newMutationTestApp(userID uuid.UUID) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userId", userID.String())
		return c.Next()
	})
	app.Put("/documents/:id/title", RequireMutation(), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusNoContent)
	})
	return app
}

func claimLeaseRequest(t *testing.T, app *fiber.App, documentID uuid.UUID, deviceID, leaseToken string, takeOver bool) *http.Response {
	t.Helper()
	body, err := json.Marshal(claimRequest{DeviceID: deviceID, LeaseToken: leaseToken, TakeOver: takeOver})
	if err != nil {
		t.Fatalf("marshal claim body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/documents/"+documentID.String()+"/lease", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("claim request: %v", err)
	}
	return response
}

func TestClaimHandlerLocksSecondDeviceAndAllowsExplicitTakeover(t *testing.T) {
	db, ownerID, documentID := setupManagerTest(t)
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })
	app := newLeaseTestApp(ownerID)

	firstResponse := claimLeaseRequest(t, app, documentID, "device-one", "", false)
	if firstResponse.StatusCode != http.StatusOK {
		t.Fatalf("first claim status = %d", firstResponse.StatusCode)
	}
	var first Grant
	if err := json.NewDecoder(firstResponse.Body).Decode(&first); err != nil {
		t.Fatalf("decode first grant: %v", err)
	}

	lockedResponse := claimLeaseRequest(t, app, documentID, "device-two", "", false)
	if lockedResponse.StatusCode != http.StatusLocked {
		t.Fatalf("second device status = %d", lockedResponse.StatusCode)
	}
	var locked errorResponse
	if err := json.NewDecoder(lockedResponse.Body).Decode(&locked); err != nil {
		t.Fatalf("decode locked response: %v", err)
	}
	if locked.Code != "EDIT_LEASE_HELD" || locked.ExpiresAt == nil {
		t.Fatalf("unexpected locked response: %+v", locked)
	}

	takeoverResponse := claimLeaseRequest(t, app, documentID, "device-two", "", true)
	if takeoverResponse.StatusCode != http.StatusOK {
		t.Fatalf("takeover status = %d", takeoverResponse.StatusCode)
	}
	if _, err := New(db).Authorize(ownerID, documentID, first.Token); !errors.Is(err, ErrLeaseInvalid) {
		t.Fatalf("first token should be invalid after takeover, got %v", err)
	}
}

func TestRequireMutationRequiresActiveLease(t *testing.T) {
	db, ownerID, documentID := setupManagerTest(t)
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })
	app := newMutationTestApp(ownerID)

	request := func(token string) *http.Response {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, "/documents/"+documentID.String()+"/title", nil)
		if token != "" {
			req.Header.Set(HeaderName, token)
		}
		response, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("mutation request: %v", err)
		}
		return response
	}

	if response := request(""); response.StatusCode != http.StatusConflict {
		t.Fatalf("missing lease status = %d", response.StatusCode)
	}
	grant, err := New(db).Claim(ownerID, documentID, "device-one", "", false)
	if err != nil {
		t.Fatalf("claim lease: %v", err)
	}
	if response := request(grant.Token); response.StatusCode != http.StatusNoContent {
		t.Fatalf("active lease status = %d", response.StatusCode)
	}
}
