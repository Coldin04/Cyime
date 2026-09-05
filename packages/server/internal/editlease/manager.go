package editlease

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"g.co1d.in/Coldin04/Cyime/server/internal/acl"
	"g.co1d.in/Coldin04/Cyime/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	DefaultTTL = 60 * time.Second
	HeaderName = "X-Cyime-Edit-Lease"
)

var (
	ErrLeaseHeld    = errors.New("document is being edited on another device")
	ErrLeaseInvalid = errors.New("edit lease is missing, expired, or replaced")
	ErrDeviceID     = errors.New("deviceId must be between 8 and 200 characters")
)

type LeaseHeldError struct {
	ExpiresAt time.Time
}

func (e *LeaseHeldError) Error() string { return ErrLeaseHeld.Error() }
func (e *LeaseHeldError) Unwrap() error { return ErrLeaseHeld }

type Grant struct {
	Token     string    `json:"leaseToken"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type Manager struct {
	db  *gorm.DB
	ttl time.Duration
	now func() time.Time
}

func New(db *gorm.DB) *Manager {
	return &Manager{db: db, ttl: DefaultTTL, now: time.Now}
}

func (m *Manager) Claim(ownerUserID, documentID uuid.UUID, deviceID, currentToken string, takeOver bool) (*Grant, error) {
	deviceID = strings.TrimSpace(deviceID)
	if len(deviceID) < 8 || len(deviceID) > 200 {
		return nil, ErrDeviceID
	}

	var grant *Grant
	err := m.db.Transaction(func(tx *gorm.DB) error {
		if _, _, err := acl.CanAccessDocumentOwnerOnly(tx, ownerUserID, documentID); err != nil {
			return err
		}

		now := m.now().UTC()
		expiresAt := now.Add(m.ttl)
		deviceHash := hashValue(deviceID)

		var current models.DocumentEditLease
		err := tx.Where("document_id = ?", documentID).Take(&current).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		leaseNotFound := errors.Is(err, gorm.ErrRecordNotFound)

		if err == nil && !takeOver && current.ExpiresAt.After(now) && current.OwnerUserID == ownerUserID {
			if current.DeviceIDHash != deviceHash {
				return &LeaseHeldError{ExpiresAt: current.ExpiresAt}
			}
			if strings.TrimSpace(currentToken) == "" || hashValue(currentToken) != current.TokenHash {
				return ErrLeaseInvalid
			}
			if err := tx.Model(&current).Updates(map[string]any{
				"expires_at": expiresAt,
				"updated_at": now,
			}).Error; err != nil {
				return err
			}
			grant = &Grant{Token: currentToken, ExpiresAt: expiresAt}
			return nil
		}

		if err == nil && !takeOver && current.OwnerUserID == ownerUserID && current.DeviceIDHash == deviceHash &&
			strings.TrimSpace(currentToken) != "" && hashValue(currentToken) == current.TokenHash {
			if err := tx.Model(&current).Updates(map[string]any{
				"expires_at": expiresAt,
				"updated_at": now,
			}).Error; err != nil {
				return err
			}
			grant = &Grant{Token: currentToken, ExpiresAt: expiresAt}
			return nil
		}

		token := strings.TrimSpace(currentToken)
		if token == "" {
			token, err = newToken()
			if err != nil {
				return err
			}
		}
		lease := models.DocumentEditLease{
			DocumentID: documentID, OwnerUserID: ownerUserID, DeviceIDHash: deviceHash,
			TokenHash: hashValue(token), ExpiresAt: expiresAt, CreatedAt: now, UpdatedAt: now,
		}
		if leaseNotFound {
			if err := tx.Create(&lease).Error; err != nil {
				return err
			}
		} else if err := tx.Model(&current).Updates(map[string]any{
			"owner_user_id":  ownerUserID,
			"device_id_hash": deviceHash,
			"token_hash":     hashValue(token),
			"expires_at":     expiresAt,
			"updated_at":     now,
		}).Error; err != nil {
			return err
		}

		grant = &Grant{Token: token, ExpiresAt: expiresAt}
		return nil
	})
	if err == nil && grant != nil {
		// Notify any WebSocket connections watching this document immediately,
		// rather than making them wait for their next renewal poll to fail.
		defaultHub.Publish(documentID, grant.Token)
	}
	return grant, err
}

func (m *Manager) Renew(ownerUserID, documentID uuid.UUID, token string) (*Grant, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrLeaseInvalid
	}

	if _, _, err := acl.CanAccessDocumentOwnerOnly(m.db, ownerUserID, documentID); err != nil {
		return nil, err
	}

	now := m.now().UTC()
	expiresAt := now.Add(m.ttl)
	result := m.db.Model(&models.DocumentEditLease{}).
		Where("document_id = ? AND owner_user_id = ? AND token_hash = ?", documentID, ownerUserID, hashValue(token)).
		Updates(map[string]any{"expires_at": expiresAt, "updated_at": now})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrLeaseInvalid
	}
	return &Grant{Token: token, ExpiresAt: expiresAt}, nil
}

func (m *Manager) Authorize(ownerUserID, documentID uuid.UUID, token string) (*models.Document, error) {
	document, _, err := acl.CanAccessDocumentOwnerOnly(m.db, ownerUserID, documentID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(token) == "" {
		return nil, ErrLeaseInvalid
	}

	var count int64
	if err := m.db.Model(&models.DocumentEditLease{}).
		Where("document_id = ? AND owner_user_id = ? AND token_hash = ? AND expires_at > ?", documentID, ownerUserID, hashValue(token), m.now().UTC()).
		Count(&count).Error; err != nil {
		return nil, err
	}
	if count != 1 {
		return nil, ErrLeaseInvalid
	}
	return document, nil
}

func (m *Manager) Release(ownerUserID, documentID uuid.UUID, token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	return m.db.Where(
		"document_id = ? AND owner_user_id = ? AND token_hash = ?",
		documentID,
		ownerUserID,
		hashValue(token),
	).Delete(&models.DocumentEditLease{}).Error
}

func newToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashValue(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
