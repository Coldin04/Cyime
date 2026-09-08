package workspace

import (
	"errors"
	"time"

	"g.co1d.in/Coldin04/Cyime/server/internal/acl"
	"g.co1d.in/Coldin04/Cyime/server/internal/database"
	"g.co1d.in/Coldin04/Cyime/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TransferDocumentOwnershipResult struct {
	DocumentID      uuid.UUID `json:"documentId"`
	PreviousOwnerID uuid.UUID `json:"previousOwnerId"`
	NewOwnerID      uuid.UUID `json:"newOwnerId"`
}

// TransferDocumentOwnership atomically moves an asset-free document to an
// existing viewer. Managed assets need a separate binary migration before the
// document owner can change without breaking future saves.
func TransferDocumentOwnership(ownerUserID, documentID, newOwnerUserID uuid.UUID) (*TransferDocumentOwnershipResult, error) {
	if ownerUserID == newOwnerUserID {
		return nil, ErrOwnershipTransferSelf
	}

	result := &TransferDocumentOwnershipResult{
		DocumentID: documentID, PreviousOwnerID: ownerUserID, NewOwnerID: newOwnerUserID,
	}
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		document, _, err := acl.CanAccessDocumentOwnerOnly(tx, ownerUserID, documentID)
		if err != nil {
			return ErrDocumentNotFoundOrUnauthorized
		}

		var targetPermission models.DocumentPermission
		if err := tx.Where("document_id = ? AND user_id = ? AND deleted_at IS NULL", documentID, newOwnerUserID).
			Take(&targetPermission).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOwnershipTransferTarget
			}
			return err
		}
		if !acl.RoleAllowsAction(targetPermission.Role, acl.ActionRead) {
			return ErrOwnershipTransferTarget
		}

		var managedAssetRefCount int64
		if err := tx.Model(&models.DocumentAssetRef{}).
			Where("document_id = ? AND deleted_at IS NULL", documentID).
			Count(&managedAssetRefCount).Error; err != nil {
			return err
		}
		if managedAssetRefCount > 0 {
			return ErrOwnershipTransferManagedAssets
		}

		if err := ensureDocumentQuotaWithinLimit(tx, newOwnerUserID, 1); err != nil {
			return err
		}
		var body models.DocumentBody
		if err := tx.Select("content_json").Where("document_id = ?", documentID).Take(&body).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := ensureWorkspaceStorageWithinLimit(tx, newOwnerUserID, int64(len(body.ContentJSON))); err != nil {
			return err
		}

		now := time.Now()
		if err := tx.Model(document).Updates(map[string]any{
			"owner_user_id":             newOwnerUserID,
			"folder_id":                 nil,
			"preferred_image_target_id": DefaultPreferredImageTargetID,
			"updated_by":                ownerUserID,
			"updated_at":                now,
		}).Error; err != nil {
			return err
		}

		if err := tx.Unscoped().Where("document_id = ? AND user_id = ?", documentID, newOwnerUserID).
			Delete(&models.DocumentPermission{}).Error; err != nil {
			return err
		}
		if err := upsertViewerPermission(tx, documentID, ownerUserID, newOwnerUserID, now); err != nil {
			return err
		}
		if err := tx.Unscoped().Where("document_id = ?", documentID).
			Delete(&models.DocumentImageTargetPreference{}).Error; err != nil {
			return err
		}
		if err := tx.Where("document_id = ?", documentID).Delete(&models.DocumentEditLease{}).Error; err != nil {
			return err
		}
		return tx.Model(&models.DocumentInvite{}).
			Where("document_id = ? AND status = ?", documentID, documentInviteStatusSent).
			Updates(map[string]any{"status": documentInviteStatusCanceled, "updated_at": now}).Error
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func upsertViewerPermission(tx *gorm.DB, documentID, userID, createdBy uuid.UUID, now time.Time) error {
	var permission models.DocumentPermission
	err := tx.Unscoped().Where("document_id = ? AND user_id = ?", documentID, userID).Take(&permission).Error
	if err == nil {
		return tx.Unscoped().Model(&permission).Updates(map[string]any{
			"role": acl.RoleViewer, "created_by": createdBy, "deleted_at": nil, "updated_at": now,
		}).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return tx.Create(&models.DocumentPermission{
		ID: uuid.New(), DocumentID: documentID, UserID: userID, Role: acl.RoleViewer, CreatedBy: createdBy,
	}).Error
}
