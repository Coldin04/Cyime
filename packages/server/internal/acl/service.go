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
	type documentRoleRow struct {
		models.Document
		PermissionRole *string `gorm:"column:permission_role"`
	}

	var row documentRoleRow
	if err := tx.
		Table("documents").
		Select("documents.*", "perms.role AS permission_role").
		Joins(
			"LEFT JOIN document_permissions AS perms ON perms.document_id = documents.id AND perms.user_id = ? AND perms.deleted_at IS NULL",
			userID,
		).
		Where("documents.id = ? AND documents.deleted_at IS NULL", documentID).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", errors.New("文档不存在或无权访问")
		}
		return nil, "", err
	}

	if row.OwnerUserID == userID {
		document := row.Document
		return &document, RoleOwner, nil
	}

	if row.PermissionRole == nil || *row.PermissionRole == "" {
		return nil, "", errors.New("文档不存在或无权访问")
	}

	document := row.Document
	return &document, *row.PermissionRole, nil
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
