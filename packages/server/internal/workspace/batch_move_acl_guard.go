package workspace

import (
	"errors"

	"g.co1d.in/Coldin04/Cyime/server/internal/acl"
	"g.co1d.in/Coldin04/Cyime/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const batchMoveUnauthorizedReason = "资源不存在或无权操作"

func normalizeBatchMoveItems(items []ItemToMove) ([]uuid.UUID, []uuid.UUID, []FailedItem) {
	folders := make([]uuid.UUID, 0, len(items))
	documents := make([]uuid.UUID, 0, len(items))
	failed := make([]FailedItem, 0)
	seenFolders := make(map[uuid.UUID]struct{})
	seenDocuments := make(map[uuid.UUID]struct{})

	for _, item := range items {
		switch item.Type {
		case "folder":
			if _, exists := seenFolders[item.ID]; exists {
				continue
			}
			seenFolders[item.ID] = struct{}{}
			folders = append(folders, item.ID)
		case "document":
			if _, exists := seenDocuments[item.ID]; exists {
				continue
			}
			seenDocuments[item.ID] = struct{}{}
			documents = append(documents, item.ID)
		default:
			failed = append(failed, FailedItem{
				ID:     item.ID,
				Type:   item.Type,
				Reason: "无效的文件类型",
			})
		}
	}

	return folders, documents, failed
}

func validateBatchMoveDestination(tx *gorm.DB, userID uuid.UUID, destFolderID *uuid.UUID) error {
	if destFolderID == nil {
		return nil
	}

	var destFolder models.Folder
	if err := tx.Where("id = ? AND owner_user_id = ? AND deleted_at IS NULL", destFolderID, userID).First(&destFolder).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("目标文件夹不存在或已被删除")
		}
		return err
	}

	return nil
}

func collectAuthorizedFoldersForMove(tx *gorm.DB, userID uuid.UUID, folderIDs []uuid.UUID) (map[uuid.UUID]models.Folder, map[uuid.UUID]struct{}, []FailedItem, error) {
	folderMap := make(map[uuid.UUID]models.Folder, len(folderIDs))
	excluded := make(map[uuid.UUID]struct{})
	failed := make([]FailedItem, 0)
	if len(folderIDs) == 0 {
		return folderMap, excluded, failed, nil
	}

	var folders []models.Folder
	if err := tx.Where("id IN ? AND owner_user_id = ? AND deleted_at IS NULL", folderIDs, userID).Find(&folders).Error; err != nil {
		return nil, nil, nil, err
	}
	for _, folder := range folders {
		folderMap[folder.ID] = folder
	}

	for _, folderID := range folderIDs {
		if _, ok := folderMap[folderID]; ok {
			continue
		}
		excluded[folderID] = struct{}{}
		failed = append(failed, FailedItem{
			ID:     folderID,
			Type:   "folder",
			Reason: batchMoveUnauthorizedReason,
		})
	}

	return folderMap, excluded, failed, nil
}

func collectAuthorizedDocumentsForMove(tx *gorm.DB, userID uuid.UUID, documentIDs []uuid.UUID) (map[uuid.UUID]models.Document, map[uuid.UUID]struct{}, []FailedItem) {
	documentMap := make(map[uuid.UUID]models.Document, len(documentIDs))
	excluded := make(map[uuid.UUID]struct{})
	failed := make([]FailedItem, 0)

	for _, documentID := range documentIDs {
		document, _, err := acl.AuthorizeDocumentAction(tx, userID, documentID, acl.ActionOwnerOnly)
		if err != nil {
			excluded[documentID] = struct{}{}
			failed = append(failed, FailedItem{
				ID:     documentID,
				Type:   "document",
				Reason: batchMoveUnauthorizedReason,
			})
			continue
		}
		documentMap[documentID] = *document
	}

	return documentMap, excluded, failed
}
