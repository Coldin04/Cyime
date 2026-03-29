package acl

import (
	"errors"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	RoleViewer       = "viewer"
	RoleEditor       = "editor"
	RoleCollaborator = "collaborator"
	RoleOwner        = "owner"
)

const (
	ActionRead          = "read"
	ActionEdit          = "edit"
	ActionManageMembers = "manage_members"
	ActionOwnerOnly     = "owner_only"
)

func ResolveDocumentRole(tx *gorm.DB, userID, documentID uuid.UUID) (*models.Document, string, error) {
	var document models.Document
	if err := tx.Where("id = ?", documentID).First(&document).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", errors.New("文档不存在或无权访问")
		}
		return nil, "", err
	}

	if document.OwnerUserID == userID {
		return &document, RoleOwner, nil
	}

	var permission models.DocumentPermission
	if err := tx.Where("document_id = ? AND user_id = ?", documentID, userID).First(&permission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", errors.New("文档不存在或无权访问")
		}
		return nil, "", err
	}

	return &document, permission.Role, nil
}

func AuthorizeDocumentAction(tx *gorm.DB, userID, documentID uuid.UUID, action string) (*models.Document, string, error) {
	document, role, err := ResolveDocumentRole(tx, userID, documentID)
	if err != nil {
		return nil, "", err
	}
	if !RoleAllowsAction(role, action) {
		return nil, "", errors.New("文档不存在或无权访问")
	}
	return document, role, nil
}

func CanReadDocument(tx *gorm.DB, userID, documentID uuid.UUID) (*models.Document, error) {
	document, _, err := AuthorizeDocumentAction(tx, userID, documentID, ActionRead)
	if err != nil {
		return nil, err
	}
	return document, nil
}

func CanEditDocument(tx *gorm.DB, userID, documentID uuid.UUID) (*models.Document, error) {
	document, _, err := AuthorizeDocumentAction(tx, userID, documentID, ActionEdit)
	if err != nil {
		return nil, err
	}
	return document, nil
}

func CanManageDocumentMembers(tx *gorm.DB, userID, documentID uuid.UUID) (*models.Document, string, error) {
	return AuthorizeDocumentAction(tx, userID, documentID, ActionManageMembers)
}

func CanAccessDocumentOwnerOnly(tx *gorm.DB, userID, documentID uuid.UUID) (*models.Document, string, error) {
	return AuthorizeDocumentAction(tx, userID, documentID, ActionOwnerOnly)
}

func CanAccessDocumentOwnerOnlyUnscoped(tx *gorm.DB, userID, documentID uuid.UUID) (*models.Document, error) {
	var document models.Document
	if err := tx.Unscoped().Where("id = ?", documentID).First(&document).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("文档不存在或无权访问")
		}
		return nil, err
	}
	if document.OwnerUserID != userID {
		return nil, errors.New("文档不存在或无权访问")
	}
	return &document, nil
}

func RoleAllowsAction(role, action string) bool {
	switch action {
	case ActionRead:
		return role == RoleOwner || role == RoleCollaborator || role == RoleEditor || role == RoleViewer
	case ActionEdit:
		return role == RoleOwner || role == RoleCollaborator || role == RoleEditor
	case ActionManageMembers:
		return role == RoleOwner || role == RoleCollaborator
	case ActionOwnerOnly:
		return role == RoleOwner
	default:
		return false
	}
}

func AllowedRolesForAction(action string) []string {
	switch action {
	case ActionRead:
		return []string{RoleViewer, RoleEditor, RoleCollaborator, RoleOwner}
	case ActionEdit:
		return []string{RoleEditor, RoleCollaborator, RoleOwner}
	case ActionManageMembers:
		return []string{RoleCollaborator, RoleOwner}
	case ActionOwnerOnly:
		return []string{RoleOwner}
	default:
		return []string{}
	}
}
