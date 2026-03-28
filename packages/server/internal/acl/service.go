package acl

import (
	"errors"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	RoleViewer = "viewer"
	RoleEditor = "editor"
	RoleOwner  = "owner"
)

func CanReadDocument(tx *gorm.DB, userID, documentID uuid.UUID) (*models.Document, error) {
	return getAccessibleDocument(tx, userID, documentID, RoleViewer)
}

func CanEditDocument(tx *gorm.DB, userID, documentID uuid.UUID) (*models.Document, error) {
	return getAccessibleDocument(tx, userID, documentID, RoleEditor)
}

func getAccessibleDocument(tx *gorm.DB, userID, documentID uuid.UUID, minRole string) (*models.Document, error) {
	var document models.Document
	if err := tx.Where("id = ? AND deleted_at IS NULL", documentID).First(&document).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("文档不存在或无权访问")
		}
		return nil, err
	}
	if document.OwnerUserID == userID {
		return &document, nil
	}

	var permission models.DocumentPermission
	if err := tx.Where("document_id = ? AND user_id = ? AND deleted_at IS NULL", documentID, userID).First(&permission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("文档不存在或无权访问")
		}
		return nil, err
	}
	if !roleAllows(permission.Role, minRole) {
		return nil, errors.New("文档不存在或无权访问")
	}
	return &document, nil
}

func roleAllows(role string, minRole string) bool {
	return roleRank(role) >= roleRank(minRole)
}

func roleRank(role string) int {
	switch role {
	case RoleOwner:
		return 3
	case RoleEditor:
		return 2
	case RoleViewer:
		return 1
	default:
		return 0
	}
}
