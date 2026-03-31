package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/acl"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/content"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/user"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ReservedFolderNames contains folder names that are reserved for system use
var ReservedFolderNames = []string{
	"回收站",
	"trash",
	".trash",
	"recycle",
	".recycle",
	"bin",
	".bin",
	"deleted",
	".deleted",
}

var ErrDocumentQuotaExceeded = errors.New("已达到文档数量上限")

const (
	DefaultPreferredImageTargetID = "managed-r2"
	LegacyPreferredImageTargetID  = "managed-r2"
	PublicAccessPrivate           = "private"
	PublicAccessAuthenticated     = "authenticated"
	PublicAccessGlobal            = "public"
)

func normalizePreferredImageTargetID(value string) string {
	trimmed := strings.TrimSpace(value)
	switch trimmed {
	case "":
		return DefaultPreferredImageTargetID
	case "managed-r2":
		return trimmed
	default:
		if _, err := uuid.Parse(trimmed); err == nil {
			return trimmed
		}
		return ""
	}
}

func resolveDocumentPreferredImageTargetID(value string) string {
	normalized := normalizePreferredImageTargetID(value)
	if normalized != "" {
		return normalized
	}
	return LegacyPreferredImageTargetID
}

func resolveUsableImageTargetForUser(userID uuid.UUID, preferredImageTargetID string) (string, error) {
	normalized := normalizePreferredImageTargetID(preferredImageTargetID)
	if normalized == "" || normalized == DefaultPreferredImageTargetID {
		return DefaultPreferredImageTargetID, nil
	}

	var count int64
	if err := database.DB.Model(&models.UserImageBedConfig{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL AND is_enabled = ?", normalized, userID, true).
		Count(&count).Error; err != nil {
		return "", err
	}
	if count == 0 {
		return DefaultPreferredImageTargetID, nil
	}
	return normalized, nil
}

func resolveEffectiveDocumentImageTargetID(userID uuid.UUID, document models.Document) (string, error) {
	candidate := document.PreferredImageTargetID

	var preference models.DocumentImageTargetPreference
	if err := database.DB.
		Where("document_id = ? AND user_id = ? AND deleted_at IS NULL", document.ID, userID).
		First(&preference).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
	} else {
		candidate = preference.TargetID
	}

	return resolveUsableImageTargetForUser(userID, candidate)
}

func resolveDocumentListExcerpt(autoExcerpt, manualExcerpt string) string {
	trimmed := strings.TrimSpace(manualExcerpt)
	if trimmed != "" {
		return trimmed
	}
	return autoExcerpt
}

func normalizePermissionRole(role string) string {
	switch strings.TrimSpace(role) {
	case acl.RoleViewer, acl.RoleEditor, acl.RoleCollaborator:
		return strings.TrimSpace(role)
	default:
		return ""
	}
}

func normalizePublicAccess(value string) string {
	switch strings.TrimSpace(value) {
	case "":
		return PublicAccessPrivate
	case PublicAccessPrivate, PublicAccessAuthenticated, PublicAccessGlobal:
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func buildDocumentPublicURL(documentID uuid.UUID) string {
	return "/view/documents/" + documentID.String()
}

func ShareDocument(actorUserID, documentID, targetUserID uuid.UUID, role string) (*ShareDocumentResponse, error) {
	normalizedRole := normalizePermissionRole(role)
	if normalizedRole == "" {
		return nil, ErrInvalidShareRole
	}
	if actorUserID == targetUserID {
		return nil, ErrCannotShareSelf
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := ensureSharingEnabledForUser(tx, actorUserID); err != nil {
			return err
		}
		_, actorRole, err := loadShareManagedDocument(tx, actorUserID, documentID)
		if err != nil {
			return err
		}
		if actorRole == acl.RoleCollaborator && normalizedRole == acl.RoleCollaborator {
			return ErrCollaboratorGrantRestricted
		}

		var targetUser models.User
		if err := tx.Select("id", "email_verified").Where("id = ?", targetUserID).First(&targetUser).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTargetUserNotFound
			}
			return err
		}
		if !targetUser.EmailVerified {
			return ErrTargetUserEmailUnverified
		}

		var permission models.DocumentPermission
		// Include soft-deleted permissions so re-sharing can "revive" an existing row
		// instead of failing unique(document_id, user_id).
		permissionErr := tx.Unscoped().Where("document_id = ? AND user_id = ?", documentID, targetUserID).First(&permission).Error
		switch {
		case permissionErr == nil:
			now := time.Now()
			if err := tx.Unscoped().Model(&models.DocumentPermission{}).
				Where("id = ?", permission.ID).
				Update("deleted_at", nil).Error; err != nil {
				return err
			}
			return tx.Unscoped().Model(&models.DocumentPermission{}).
				Where("id = ?", permission.ID).
				Updates(map[string]any{
					"role":       normalizedRole,
					"updated_at": now,
				}).Error
		case errors.Is(permissionErr, gorm.ErrRecordNotFound):
			return tx.Create(&models.DocumentPermission{
				ID:         uuid.New(),
				DocumentID: documentID,
				UserID:     targetUserID,
				Role:       normalizedRole,
				CreatedBy:  actorUserID,
			}).Error
		default:
			return permissionErr
		}
	})
	if err != nil {
		return nil, err
	}

	return ListDocumentMembers(actorUserID, documentID)
}

func RemoveDocumentMember(actorUserID, documentID, targetUserID uuid.UUID) (*ShareDocumentResponse, error) {
	if actorUserID == targetUserID {
		return nil, ErrCannotRemoveSelf
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := ensureSharingEnabledForUser(tx, actorUserID); err != nil {
			return err
		}
		document, actorRole, err := loadShareManagedDocument(tx, actorUserID, documentID)
		if err != nil {
			return err
		}
		if document.OwnerUserID == targetUserID {
			return ErrCannotRemoveOwner
		}

		var targetPermission models.DocumentPermission
		if err := tx.Where("document_id = ? AND user_id = ? AND deleted_at IS NULL", documentID, targetUserID).First(&targetPermission).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrMemberNotFound
			}
			return err
		}
		if actorRole == acl.RoleCollaborator && targetPermission.Role == acl.RoleCollaborator {
			return ErrCollaboratorRemoveRestricted
		}

		return tx.Where("document_id = ? AND user_id = ?", documentID, targetUserID).Delete(&models.DocumentPermission{}).Error
	})
	if err != nil {
		return nil, err
	}

	return ListDocumentMembers(actorUserID, documentID)
}

func LeaveSharedDocument(userID, documentID uuid.UUID) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		document, role, err := acl.AuthorizeDocumentAction(tx, userID, documentID, acl.ActionRead)
		if err != nil {
			return ErrDocumentNotFoundOrUnauthorized
		}
		if role == acl.RoleOwner || document.OwnerUserID == userID {
			return ErrOwnerCannotLeaveShared
		}

		result := tx.Where("document_id = ? AND user_id = ? AND deleted_at IS NULL", documentID, userID).Delete(&models.DocumentPermission{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrDocumentNotFoundOrUnauthorized
		}
		return nil
	})
}

func ListDocumentMembers(actorUserID, documentID uuid.UUID) (*ShareDocumentResponse, error) {
	document, _, err := acl.AuthorizeDocumentAction(database.DB, actorUserID, documentID, acl.ActionRead)
	if err != nil {
		return nil, ErrDocumentNotFoundOrUnauthorized
	}

	var permissions []models.DocumentPermission
	if err := database.DB.Where("document_id = ? AND deleted_at IS NULL", documentID).Order("created_at asc").Find(&permissions).Error; err != nil {
		return nil, err
	}

	userIDs := make([]uuid.UUID, 0, len(permissions)+1)
	userIDs = append(userIDs, document.OwnerUserID)
	for _, permission := range permissions {
		userIDs = append(userIDs, permission.UserID)
	}

	var users []models.User
	if err := database.DB.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	userMap := make(map[uuid.UUID]models.User, len(users))
	for _, user := range users {
		userMap[user.ID] = user
	}

	members := make([]ShareDocumentMember, 0, len(permissions)+1)
	ownerUser := userMap[document.OwnerUserID]
	members = append(members, ShareDocumentMember{
		UserID:      document.OwnerUserID,
		Role:        acl.RoleOwner,
		DisplayName: ownerUser.DisplayName,
		Email:       ownerUser.Email,
	})
	for _, permission := range permissions {
		memberUser := userMap[permission.UserID]
		members = append(members, ShareDocumentMember{
			UserID:      permission.UserID,
			Role:        permission.Role,
			DisplayName: memberUser.DisplayName,
			Email:       memberUser.Email,
		})
	}

	return &ShareDocumentResponse{
		DocumentID: documentID,
		Members:    members,
	}, nil
}

func ListSharedDocuments(userID uuid.UUID, limit, offset int) (*SharedDocumentListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := database.DB.
		Table("document_permissions AS perms").
		Joins("JOIN documents AS d ON d.id = perms.document_id AND d.deleted_at IS NULL").
		Where("perms.user_id = ? AND perms.deleted_at IS NULL", userID).
		Where("perms.role IN ?", acl.AllowedRolesForAction(acl.ActionRead))

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	type row struct {
		DocumentID             uuid.UUID
		Title                  string
		Excerpt                string
		ManualExcerpt          string
		DocumentType           string
		PreferredImageTargetID string
		FolderID               *uuid.UUID
		OwnerUserID            uuid.UUID
		MyRole                 string
		CreatedAt              time.Time
		UpdatedAt              time.Time
	}

	var rows []row
	if err := query.
		Select("d.id AS document_id", "d.title", "d.excerpt", "d.manual_excerpt", "d.document_type", "d.preferred_image_target_id", "d.folder_id", "d.owner_user_id", "perms.role AS my_role", "d.created_at", "d.updated_at").
		Order("d.updated_at desc").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	ownerIDs := make([]uuid.UUID, 0, len(rows))
	for _, item := range rows {
		ownerIDs = append(ownerIDs, item.OwnerUserID)
	}

	var owners []models.User
	if len(ownerIDs) > 0 {
		if err := database.DB.Where("id IN ?", ownerIDs).Find(&owners).Error; err != nil {
			return nil, err
		}
	}
	ownerMap := make(map[uuid.UUID]models.User, len(owners))
	for _, owner := range owners {
		ownerMap[owner.ID] = owner
	}

	items := make([]SharedDocumentItem, 0, len(rows))
	for _, item := range rows {
		owner := ownerMap[item.OwnerUserID]
		items = append(items, SharedDocumentItem{
			DocumentID:             item.DocumentID,
			Title:                  item.Title,
			Excerpt:                resolveDocumentListExcerpt(item.Excerpt, item.ManualExcerpt),
			DocumentType:           item.DocumentType,
			PreferredImageTargetID: resolveDocumentPreferredImageTargetID(item.PreferredImageTargetID),
			FolderID:               item.FolderID,
			OwnerUserID:            item.OwnerUserID,
			OwnerDisplayName:       owner.DisplayName,
			MyRole:                 item.MyRole,
			CreatedAt:              item.CreatedAt,
			UpdatedAt:              item.UpdatedAt,
		})
	}

	return &SharedDocumentListResponse{
		Items:   items,
		HasMore: int64(offset+len(items)) < total,
		Total:   total,
	}, nil
}

func GetSharedDocumentSummary(userID uuid.UUID) (*SharedDocumentSummaryResponse, error) {
	type probeRow struct {
		DocumentID uuid.UUID
	}

	var row probeRow
	err := database.DB.
		Table("document_permissions AS perms").
		Joins("JOIN documents AS d ON d.id = perms.document_id AND d.deleted_at IS NULL").
		Where("perms.user_id = ? AND perms.deleted_at IS NULL", userID).
		Where("perms.role IN ?", acl.AllowedRolesForAction(acl.ActionRead)).
		Select("perms.document_id").
		Limit(1).
		Take(&row).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return &SharedDocumentSummaryResponse{HasSharedDocuments: false}, nil
	case err != nil:
		return nil, err
	default:
		return &SharedDocumentSummaryResponse{HasSharedDocuments: true}, nil
	}
}

// GetFiles retrieves a list of files (folders and documents) for a given user and parent folder
func GetFiles(userID uuid.UUID, parentID *uuid.UUID, limit, offset int, sortBy, order, filterType string) (*FileListResponse, error) {
	// Default values
	if limit <= 0 {
		limit = 50
	}
	if sortBy == "" {
		sortBy = "updated_at"
	}
	if order == "" {
		order = "desc"
	}
	if filterType == "" {
		filterType = "all"
	}

	// Validate sort_by field
	validSorts := map[string]bool{
		"name": true, "title": true, "created_at": true, "updated_at": true,
	}
	if !validSorts[sortBy] {
		sortBy = "updated_at"
	}
	validSortFields := map[string]bool{
		"name":       true,
		"updated_at": true,
		"created_at": true,
	}
	if !validSortFields[sortBy] {
		sortBy = "updated_at"
	}

	// Validate order
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	var items []FileItem
	var total int64

	// Build the query based on filter type
	if filterType == "folders" {
		// Only folders
		query := database.DB.Model(&models.Folder{}).Where("owner_user_id = ? AND deleted_at IS NULL", userID)
		if parentID != nil {
			query = query.Where("parent_id = ?", parentID)
		}

		query.Count(&total)

		var folders []models.Folder
		if err := query.Order(sortBy + " " + order).Limit(limit).Offset(offset).Find(&folders).Error; err != nil {
			return nil, err
		}

		for _, f := range folders {
			items = append(items, folderToFileItem(f))
		}

	} else if filterType == "documents" {
		// Only documents
		query := database.DB.Model(&models.Document{}).Where("owner_user_id = ? AND deleted_at IS NULL", userID)
		if parentID != nil {
			query = query.Where("folder_id = ?", parentID)
		} else {
			query = query.Where("folder_id IS NULL")
		}

		query.Count(&total)

		var documents []models.Document
		if err := query.Select("id", "owner_user_id", "folder_id", "title", "excerpt", "manual_excerpt", "document_type", "created_at", "updated_at", "created_by").Order(sortBy + " " + order).Limit(limit).Offset(offset).Find(&documents).Error; err != nil {
			return nil, err
		}

		for _, m := range documents {
			items = append(items, documentToFileItem(m, ""))
		}

	} else {
		// UNION ALL for both folders and documents
		// Count total for both types
		var folderCount, documentCount int64

		folderQuery := database.DB.Model(&models.Folder{}).Where("owner_user_id = ? AND deleted_at IS NULL", userID)
		if parentID != nil {
			folderQuery = folderQuery.Where("parent_id = ?", parentID)
		} else {
			folderQuery = folderQuery.Where("parent_id IS NULL")
		}
		folderQuery.Count(&folderCount)

		documentQuery := database.DB.Model(&models.Document{}).Where("owner_user_id = ? AND deleted_at IS NULL", userID)
		if parentID != nil {
			documentQuery = documentQuery.Where("folder_id = ?", parentID)
		} else {
			documentQuery = documentQuery.Where("folder_id IS NULL")
		}
		documentQuery.Count(&documentCount)

		total = folderCount + documentCount

		// Fetch folders
		var folders []models.Folder
		folderQuery.Order(sortBy + " " + order).Limit(limit).Offset(offset).Find(&folders)

		for _, f := range folders {
			items = append(items, folderToFileItem(f))
		}

		// Fetch documents (adjust limit to account for already fetched folders)
		remainingLimit := limit - len(folders)
		if remainingLimit > 0 {
			var documents []models.Document
			documentQuery.Order(sortBy + " " + order).Limit(remainingLimit).Offset(max(0, offset-len(folders))).Find(&documents)

			for _, m := range documents {
				items = append(items, documentToFileItem(m, ""))
			}
		}
	}

	return &FileListResponse{
		Items:   items,
		HasMore: int64(offset+len(items)) < total,
		Total:   total,
	}, nil
}

// CreateFolder creates a new folder with validation
func CreateFolder(userID uuid.UUID, name string, description *string, parentID *uuid.UUID) (*models.Folder, error) {
	// Validate name is not empty
	if strings.TrimSpace(name) == "" {
		return nil, ErrFolderNameRequired
	}

	// Validate name length
	if len(name) > 255 {
		return nil, ErrFolderNameTooLong
	}

	// Validate not a reserved name
	lowerName := strings.ToLower(strings.TrimSpace(name))
	for _, reserved := range ReservedFolderNames {
		if lowerName == strings.ToLower(reserved) {
			return nil, ErrReservedFolderName
		}
	}

	// Validate parent folder exists if provided
	if parentID != nil {
		var parent models.Folder
		result := database.DB.Where("id = ? AND owner_user_id = ?", parentID, userID).First(&parent)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, ErrParentFolderNotFound
			}
			return nil, result.Error
		}

		// Validate no duplicate name in same parent
		var existing models.Folder
		result = database.DB.Where("owner_user_id = ? AND parent_id = ? AND name = ? AND deleted_at IS NULL", userID, parentID, name).First(&existing)
		if result.Error == nil {
			return nil, ErrDuplicateFolderName
		}
	} else {
		// Validate no duplicate name in root
		var existing models.Folder
		result := database.DB.Where("owner_user_id = ? AND parent_id IS NULL AND name = ? AND deleted_at IS NULL", userID, name).First(&existing)
		if result.Error == nil {
			return nil, ErrDuplicateFolderName
		}
	}

	// Create the folder
	folder := &models.Folder{
		ID:          uuid.New(),
		OwnerUserID: userID,
		ParentID:    parentID,
		Name:        name,
		Description: description,
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}

	if err := database.DB.Create(folder).Error; err != nil {
		return nil, err
	}

	return folder, nil
}

// CreateDocument creates a new document with unique title handling.
func CreateDocument(userID uuid.UUID, title string, contentJSON string, folderID *uuid.UUID, documentType string, preferredImageTargetID string) (*models.Document, error) {
	if documentType == "" {
		documentType = "rich_text"
	}
	if documentType != "rich_text" && documentType != "table" {
		return nil, ErrUnsupportedDocumentType
	}
	if preferredImageTargetID == "" {
		preferredImageTargetID = DefaultPreferredImageTargetID
	}
	preferredImageTargetID = normalizePreferredImageTargetID(preferredImageTargetID)
	if preferredImageTargetID == "" {
		return nil, ErrUnsupportedImageTarget
	}

	// Validate title length
	if len(title) > 255 {
		return nil, ErrDocumentTitleTooLong
	}

	// Validate folder exists if provided
	if folderID != nil {
		var folder models.Folder
		result := database.DB.Where("id = ? AND owner_user_id = ?", folderID, userID).First(&folder)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, ErrFolderNotFound
			}
			return nil, result.Error
		}
	}

	newTitle := strings.TrimSpace(title)
	if newTitle == "" || newTitle == "未命名文档" {
		// Generate a unique title based on date and sequence
		datePrefix := time.Now().Format("060102")
		baseTitleWithDate := "未命名文档" + datePrefix

		var count int64
		query := database.DB.Model(&models.Document{}).Where("owner_user_id = ? AND title LIKE ? AND deleted_at IS NULL", userID, baseTitleWithDate+"-%")
		if folderID != nil {
			query = query.Where("folder_id = ?", folderID)
		} else {
			query = query.Where("folder_id IS NULL")
		}
		query.Count(&count)

		newTitle = fmt.Sprintf("%s-%02d", baseTitleWithDate, count+1)
	} else {
		// For custom titles, check for duplicates in the same folder
		var existing models.Document
		query := database.DB.Model(&models.Document{}).Where("owner_user_id = ? AND title = ? AND deleted_at IS NULL", userID, newTitle)
		if folderID != nil {
			query = query.Where("folder_id = ?", folderID)
		} else {
			query = query.Where("folder_id IS NULL")
		}
		result := query.First(&existing)
		if result.Error == nil {
			// A record was found, so it's a duplicate
			return nil, ErrDuplicateDocumentTitle
		}
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// A real database error occurred
			return nil, result.Error
		}
	}

	// Generate excerpt from canonical editor JSON.
	excerpt := content.BuildExcerptFromContentJSON(contentJSON)

	// Create the document in a transaction
	var document *models.Document
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		limit, err := user.GetEffectiveDocumentQuota(userID)
		if err != nil {
			return err
		}
		if limit != nil {
			var activeCount int64
			if err := tx.Model(&models.Document{}).
				Where("owner_user_id = ? AND deleted_at IS NULL", userID).
				Count(&activeCount).Error; err != nil {
				return err
			}
			if activeCount >= int64(*limit) {
				return ErrDocumentQuotaExceeded
			}
		}

		// Create document metadata
		document = &models.Document{
			ID:                     uuid.New(),
			OwnerUserID:            userID,
			FolderID:               folderID,
			Title:                  newTitle,
			Excerpt:                excerpt,
			DocumentType:           documentType,
			PreferredImageTargetID: preferredImageTargetID,
			EditorType:             "tiptap",
			CreatedBy:              userID,
			UpdatedBy:              userID,
		}

		if err := tx.Create(document).Error; err != nil {
			return err
		}

		if err := content.CreateInitialContent(tx, document.ID, userID, contentJSON); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return document, nil
}

// BatchDeleteFiles soft deletes multiple files (folders and documents) in a single transaction
func BatchDeleteFiles(userID uuid.UUID, itemsToDelete []ItemToDelete) (*BatchDeleteResponse, error) {
	var successCount int
	var failedItems []FailedItem

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range itemsToDelete {
			if item.Type == "folder" {
				if err := deleteFolderRecursive(tx, userID, item.ID); err != nil {
					failedItems = append(failedItems, FailedItem{
						ID:     item.ID,
						Type:   item.Type,
						Reason: err.Error(),
					})
					continue
				}
				successCount++
			} else if item.Type == "document" {
				// Document destructive operations stay owner-only in V1.
				if _, _, err := acl.AuthorizeDocumentAction(tx, userID, item.ID, acl.ActionOwnerOnly); err != nil {
					failedItems = append(failedItems, FailedItem{
						ID:     item.ID,
						Type:   item.Type,
						Reason: "文档不存在或无权删除",
					})
					continue
				}

				// Delete content first
				if err := content.DeleteContentByDocumentID(tx, userID, item.ID); err != nil {
					failedItems = append(failedItems, FailedItem{
						ID:     item.ID,
						Type:   item.Type,
						Reason: err.Error(),
					})
					continue
				}

				// Soft delete document metadata
				result := tx.Where("id = ?", item.ID).Delete(&models.Document{})
				if result.Error != nil {
					failedItems = append(failedItems, FailedItem{
						ID:     item.ID,
						Type:   item.Type,
						Reason: result.Error.Error(),
					})
					continue
				}
				if result.RowsAffected == 0 {
					failedItems = append(failedItems, FailedItem{
						ID:     item.ID,
						Type:   item.Type,
						Reason: "文档不存在或无权删除",
					})
					continue
				}
				successCount++
			} else {
				failedItems = append(failedItems, FailedItem{
					ID:     item.ID,
					Type:   item.Type,
					Reason: "无效的文件类型",
				})
			}
		}
		// Don't return error here, we want to commit even if some items fail
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &BatchDeleteResponse{
		Success:     len(failedItems) == 0,
		Message:     buildBatchDeleteMessage(successCount, len(failedItems)),
		FailedItems: failedItems,
	}, nil
}

func buildBatchDeleteMessage(successCount, failedCount int) string {
	if failedCount == 0 {
		return "已成功删除所有项目"
	}
	if successCount == 0 {
		return "删除失败"
	}
	return fmt.Sprintf("已成功删除 %d 个项目，%d 个失败", successCount, failedCount)
}

// DeleteFile soft deletes a file (folder or document)
func DeleteFile(userID uuid.UUID, fileID uuid.UUID, fileType string) error {
	if fileType == "folder" {
		// Start a single transaction for the entire recursive operation
		return database.DB.Transaction(func(tx *gorm.DB) error {
			return deleteFolderRecursive(tx, userID, fileID)
		})
	} else if fileType == "document" {
		// Start a transaction to delete document and its content
		return database.DB.Transaction(func(tx *gorm.DB) error {
			// Guard owner-only mutations early for clearer permission semantics.
			if _, _, err := acl.AuthorizeDocumentAction(tx, userID, fileID, acl.ActionOwnerOnly); err != nil {
				return errors.New("文档不存在或无权删除")
			}

			// Delete content first
			if err := content.DeleteContentByDocumentID(tx, userID, fileID); err != nil {
				return err
			}

			// Soft delete document metadata
			result := tx.Where("id = ?", fileID).Delete(&models.Document{})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return errors.New("文档不存在或无权删除")
			}
			return nil
		})
	}

	return errors.New("无效的文件类型")
}

// deleteFolderRecursive recursively deletes a folder and all its children within a single transaction
func deleteFolderRecursive(tx *gorm.DB, userID uuid.UUID, folderID uuid.UUID) error {
	// Find all child folders within the transaction
	var childFolders []models.Folder
	if err := tx.Where("parent_id = ? AND owner_user_id = ?", folderID, userID).Find(&childFolders).Error; err != nil {
		return err
	}

	// Recursively delete child folders, passing the transaction down
	for _, child := range childFolders {
		if err := deleteFolderRecursive(tx, userID, child.ID); err != nil {
			return err
		}
	}

	// Find all documents in this folder
	var documents []models.Document
	if err := tx.Where("folder_id = ? AND owner_user_id = ?", folderID, userID).Find(&documents).Error; err != nil {
		return err
	}

	// Delete content for each document, then soft delete the document
	for _, md := range documents {
		// Delete content
		if err := content.DeleteContentByDocumentID(tx, userID, md.ID); err != nil {
			return err
		}
		// Soft delete document metadata
		if err := tx.Where("id = ?", md.ID).Delete(&models.Document{}).Error; err != nil {
			return err
		}
	}

	// Soft delete the folder itself within the transaction
	result := tx.Where("id = ? AND owner_user_id = ?", folderID, userID).Delete(&models.Folder{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("文件夹不存在或无权删除")
	}

	return nil
}

// folderToFileItem converts a Folder model to FileItem DTO
func folderToFileItem(folder models.Folder) FileItem {
	return FileItem{
		ID:          folder.ID,
		Type:        "folder",
		Name:        folder.Name,
		Description: folder.Description,
		ParentID:    folder.ParentID,
		CreatedAt:   folder.CreatedAt,
		UpdatedAt:   folder.UpdatedAt,
		Creator: CreatorInfo{
			ID:          folder.CreatedBy,
			DisplayName: nil, // Will be populated if needed
		},
	}
}

// documentToFileItem converts a Document model to FileItem DTO
func documentToFileItem(document models.Document, role string) FileItem {
	preferredImageTargetID := resolveDocumentPreferredImageTargetID(document.PreferredImageTargetID)
	excerpt := resolveDocumentListExcerpt(document.Excerpt, document.ManualExcerpt)
	manualExcerpt := document.ManualExcerpt
	publicAccess := normalizePublicAccess(document.PublicAccess)
	publicURL := buildDocumentPublicURL(document.ID)
	var myRole *string
	if strings.TrimSpace(role) != "" {
		myRole = &role
	}
	return FileItem{
		ID:                     document.ID,
		Type:                   "document",
		DocumentType:           &document.DocumentType,
		PreferredImageTargetID: &preferredImageTargetID,
		MyRole:                 myRole,
		PublicAccess:           &publicAccess,
		PublicURL:              &publicURL,
		Name:                   document.Title,
		Title:                  &document.Title,
		Excerpt:                &excerpt,
		ManualExcerpt:          &manualExcerpt,
		FolderID:               document.FolderID,
		CreatedAt:              document.CreatedAt,
		UpdatedAt:              document.UpdatedAt,
		Creator: CreatorInfo{
			ID:          document.CreatedBy,
			DisplayName: nil,
		},
	}
}

// generateExcerpt generates a plain text excerpt from document content
func generateExcerpt(content string) string {
	// Simple excerpt generation: take first 100 characters of plain text
	// In a real implementation, you might want to strip document syntax
	if len(content) == 0 {
		return ""
	}

	// Remove document syntax (basic stripping)
	plainText := content
	// Remove headers
	plainText = strings.ReplaceAll(plainText, "# ", "")
	plainText = strings.ReplaceAll(plainText, "## ", "")
	plainText = strings.ReplaceAll(plainText, "### ", "")
	// Remove bold/italic
	plainText = strings.ReplaceAll(plainText, "**", "")
	plainText = strings.ReplaceAll(plainText, "*", "")
	// Remove links
	plainText = strings.ReplaceAll(plainText, "[]()", "")

	// Trim and take first 100 characters
	plainText = strings.TrimSpace(plainText)
	if len(plainText) > 100 {
		return plainText[:100] + "..."
	}

	return plainText
}

// GetFile retrieves a single file (folder or document) by ID
func GetFile(userID uuid.UUID, fileID uuid.UUID, fileType string) (*FileItem, error) {
	if fileType == "folder" {
		var folder models.Folder
		result := database.DB.Where("id = ? AND owner_user_id = ?", fileID, userID).First(&folder)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, ErrFileNotFound
			}
			return nil, result.Error
		}

		item := folderToFileItem(folder)
		return &item, nil
	} else if fileType == "document" {
		document, role, err := acl.AuthorizeDocumentAction(database.DB, userID, fileID, acl.ActionRead)
		if err != nil {
			return nil, ErrFileNotFound
		}

		item := documentToFileItem(*document, role)
		effectiveTargetID, resolveErr := resolveEffectiveDocumentImageTargetID(userID, *document)
		if resolveErr != nil {
			return nil, resolveErr
		}
		item.PreferredImageTargetID = &effectiveTargetID
		return &item, nil
	}

	return nil, errors.New("无效的文件类型")
}

func GetPublicDocument(fileID uuid.UUID, userID *uuid.UUID) (*FileItem, error) {
	var document models.Document
	if err := database.DB.
		Where("id = ? AND deleted_at IS NULL", fileID).
		First(&document).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPublicDocumentNotFound
		}
		return nil, err
	}
	switch normalizePublicAccess(document.PublicAccess) {
	case PublicAccessGlobal:
		// ok
	case PublicAccessAuthenticated:
		if userID == nil || *userID == uuid.Nil {
			return nil, ErrPublicDocumentAuthRequired
		}
	default:
		return nil, ErrPublicDocumentNotFound
	}

	item := documentToFileItem(document, "")
	return &item, nil
}

func sanitizePublicContentJSON(raw string) json.RawMessage {
	var node any
	if err := json.Unmarshal([]byte(raw), &node); err != nil {
		return json.RawMessage(raw)
	}
	stripImageAttrs(node)
	normalized, err := json.Marshal(node)
	if err != nil {
		return json.RawMessage(raw)
	}
	return normalized
}

func stripImageAttrs(node any) {
	obj, ok := node.(map[string]any)
	if !ok {
		return
	}
	if nodeType, _ := obj["type"].(string); nodeType == "image" {
		if attrs, ok := obj["attrs"].(map[string]any); ok {
			if rawSrc, ok := attrs["src"].(string); ok {
				src := strings.TrimSpace(rawSrc)
				keepExternal := strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://")
				if strings.Contains(src, "/api/v1/media/assets/") {
					keepExternal = false
				}
				if !keepExternal {
					delete(attrs, "src")
				}
			} else {
				delete(attrs, "src")
			}
			delete(attrs, "assetId")
			obj["attrs"] = attrs
		}
	}
	if children, ok := obj["content"].([]any); ok {
		for _, child := range children {
			stripImageAttrs(child)
		}
	}
}

func GetPublicDocumentContent(documentID uuid.UUID, userID *uuid.UUID) (*DocumentPublicContentResponse, error) {
	var document models.Document
	if err := database.DB.
		Select("id", "public_access").
		Where("id = ? AND deleted_at IS NULL", documentID).
		First(&document).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPublicDocumentNotFound
		}
		return nil, err
	}
	switch normalizePublicAccess(document.PublicAccess) {
	case PublicAccessGlobal:
		// ok
	case PublicAccessAuthenticated:
		if userID == nil || *userID == uuid.Nil {
			return nil, ErrPublicDocumentAuthRequired
		}
	default:
		return nil, ErrPublicDocumentNotFound
	}

	var body models.DocumentBody
	if err := database.DB.
		Where("document_id = ? AND deleted_at IS NULL", documentID).
		First(&body).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPublicDocumentNotFound
		}
		return nil, err
	}

	return &DocumentPublicContentResponse{
		ID:             body.ID,
		DocumentID:     body.DocumentID,
		ContentJSON:    sanitizePublicContentJSON(body.ContentJSON),
		PlainText:      body.PlainText,
		ContentVersion: body.ContentVersion,
		CreatedAt:      body.CreatedAt,
		UpdatedAt:      body.UpdatedAt,
	}, nil
}

// UpdateDocumentTitle updates the title of a document.
func UpdateDocumentTitle(userID uuid.UUID, documentID uuid.UUID, title string) error {
	trimmedTitle := strings.TrimSpace(title)

	// Validate title length
	if len(trimmedTitle) == 0 {
		return ErrDocumentTitleRequired
	}
	if len(trimmedTitle) > 255 {
		return ErrDocumentTitleTooLong
	}

	document, _, err := acl.AuthorizeDocumentAction(database.DB, userID, documentID, acl.ActionEdit)
	if err != nil {
		return ErrDocumentNotFoundOrUnauthorized
	}

	// Same title is treated as a no-op.
	if document.Title == trimmedTitle {
		return nil
	}

	// Check duplicate title in the same folder.
	var conflictCount int64
	query := database.DB.Model(&models.Document{}).
		Where("owner_user_id = ? AND id <> ? AND title = ? AND deleted_at IS NULL", document.OwnerUserID, documentID, trimmedTitle)
	if document.FolderID != nil {
		query = query.Where("folder_id = ?", document.FolderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}
	if err := query.Count(&conflictCount).Error; err != nil {
		return err
	}
	if conflictCount > 0 {
		return ErrDuplicateDocumentTitle
	}

	// Update the title
	return database.DB.Model(document).Update("title", trimmedTitle).Error
}

func UpdateDocumentManualExcerpt(userID uuid.UUID, documentID uuid.UUID, manualExcerpt string) (string, string, error) {
	trimmed := strings.TrimSpace(manualExcerpt)
	if len(trimmed) > 500 {
		return "", "", ErrDocumentExcerptTooLong
	}

	document, _, err := acl.AuthorizeDocumentAction(database.DB, userID, documentID, acl.ActionManageMembers)
	if err != nil {
		return "", "", ErrDocumentNotFoundOrUnauthorized
	}

	if document.ManualExcerpt == trimmed {
		return document.ManualExcerpt, resolveDocumentListExcerpt(document.Excerpt, document.ManualExcerpt), nil
	}

	if err := database.DB.Model(document).Updates(map[string]any{
		"manual_excerpt": trimmed,
		"updated_at":     time.Now(),
		"updated_by":     userID,
	}).Error; err != nil {
		return "", "", err
	}

	return trimmed, resolveDocumentListExcerpt(document.Excerpt, trimmed), nil
}

func UpdateDocumentImageTarget(userID uuid.UUID, documentID uuid.UUID, preferredImageTargetID string) error {
	document, _, err := acl.AuthorizeDocumentAction(database.DB, userID, documentID, acl.ActionEdit)
	if err != nil {
		return ErrDocumentNotFoundOrUnauthorized
	}

	normalized := normalizePreferredImageTargetID(preferredImageTargetID)
	if normalized == "" {
		return ErrUnsupportedImageTarget
	}
	if normalized != DefaultPreferredImageTargetID {
		var config models.UserImageBedConfig
		result := database.DB.
			Where("id = ? AND user_id = ? AND deleted_at IS NULL AND is_enabled = ?", normalized, userID, true).
			First(&config)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return ErrImageTargetNotFound
			}
			return result.Error
		}
	}

	var preference models.DocumentImageTargetPreference
	preferenceErr := database.DB.Unscoped().
		Where("document_id = ? AND user_id = ?", document.ID, userID).
		First(&preference).Error
	switch {
	case preferenceErr == nil:
		return database.DB.Unscoped().Model(&models.DocumentImageTargetPreference{}).
			Where("id = ?", preference.ID).
			Updates(map[string]any{
				"target_id":   normalized,
				"deleted_at":  nil,
				"updated_at":  time.Now(),
				"document_id": document.ID,
				"user_id":     userID,
			}).Error
	case errors.Is(preferenceErr, gorm.ErrRecordNotFound):
		return database.DB.Create(&models.DocumentImageTargetPreference{
			ID:         uuid.New(),
			DocumentID: document.ID,
			UserID:     userID,
			TargetID:   normalized,
		}).Error
	default:
		return preferenceErr
	}
}

// UpdateFolderName updates the name of a folder
func UpdateFolderName(userID uuid.UUID, folderID uuid.UUID, name string) error {
	trimmedName := strings.TrimSpace(name)

	// Validate name length
	if len(trimmedName) == 0 {
		return ErrFolderNameRequired
	}
	if len(trimmedName) > 255 {
		return ErrFolderNameTooLong
	}

	lowerName := strings.ToLower(trimmedName)
	for _, reserved := range ReservedFolderNames {
		if lowerName == strings.ToLower(reserved) {
			return ErrReservedFolderName
		}
	}

	// Verify the folder exists and belongs to the user
	var folder models.Folder
	result := database.DB.Where("id = ? AND owner_user_id = ?", folderID, userID).First(&folder)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return ErrFolderNotFound
		}
		return result.Error
	}

	// Same name is treated as a no-op.
	if folder.Name == trimmedName {
		return nil
	}

	var conflictCount int64
	query := database.DB.Model(&models.Folder{}).
		Where("owner_user_id = ? AND id <> ? AND name = ? AND deleted_at IS NULL", userID, folderID, trimmedName)
	if folder.ParentID != nil {
		query = query.Where("parent_id = ?", folder.ParentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}
	if err := query.Count(&conflictCount).Error; err != nil {
		return err
	}
	if conflictCount > 0 {
		return ErrDuplicateFolderName
	}

	// Update the name
	return database.DB.Model(&folder).Update("name", trimmedName).Error
}

// MoveDocument moves a document to a different folder (or root).
func MoveDocument(userID uuid.UUID, documentID uuid.UUID, folderID *uuid.UUID) (*time.Time, error) {
	// Move keeps owner-only semantics in V1: shared members can edit, but cannot reorganize owner tree.
	document, _, err := acl.AuthorizeDocumentAction(database.DB, userID, documentID, acl.ActionOwnerOnly)
	if err != nil {
		return nil, ErrDocumentNotFoundOrDeleted
	}

	// 2. Validate target folder if provided
	if folderID != nil {
		var folder models.Folder
		result := database.DB.Where("id = ? AND owner_user_id = ? AND deleted_at IS NULL", folderID, userID).First(&folder)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, ErrTargetFolderNotFoundOrDeleted
			}
			return nil, result.Error
		}
	}

	// 3. Check for naming conflict in the destination
	var conflictCount int64
	query := database.DB.Model(&models.Document{}).Where("title = ? AND owner_user_id = ? AND deleted_at IS NULL", document.Title, document.OwnerUserID)
	if folderID != nil {
		query = query.Where("folder_id = ?", folderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}
	query.Count(&conflictCount)
	if conflictCount > 0 {
		return nil, errors.New("目标文件夹中已存在同名文档")
	}

	// 4. Update the folder_id
	var updatedAt time.Time
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Update folder_id
		if err := tx.Model(&document).Update("folder_id", folderID).Error; err != nil {
			return err
		}

		// Update updated_at timestamp
		now := time.Now()
		updatedAt = now
		return tx.Model(&document).Update("updated_at", now).Error
	})

	if err != nil {
		return nil, err
	}

	return &updatedAt, nil
}

// MoveFolder moves a folder to a different parent folder (or root)
func MoveFolder(userID uuid.UUID, folderID uuid.UUID, parentID *uuid.UUID) (*time.Time, error) {
	// 1. Verify the folder exists, belongs to the user, and is not deleted
	var folder models.Folder
	result := database.DB.Where("id = ? AND owner_user_id = ? AND deleted_at IS NULL", folderID, userID).First(&folder)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrFolderNotFoundOrDeleted
		}
		return nil, result.Error
	}

	// 2. Prevent moving a folder into itself
	if parentID != nil && *parentID == folderID {
		return nil, errors.New("不能将文件夹移动到其自身内部")
	}

	// 3. Validate target parent folder if provided
	if parentID != nil {
		// Check if parent folder exists and belongs to user
		var parentFolder models.Folder
		result = database.DB.Where("id = ? AND owner_user_id = ? AND deleted_at IS NULL", parentID, userID).First(&parentFolder)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, ErrTargetParentNotFoundOrDeleted
			}
			return nil, result.Error
		}

		// 4. Check for circular dependency: cannot move folder into its own descendant
		if err := checkCircularDependency(database.DB, userID, &folderID, parentID); err != nil {
			return nil, err
		}
	}

	// 5. Check for naming conflict in the destination
	var conflictCount int64
	query := database.DB.Model(&models.Folder{}).Where("name = ? AND owner_user_id = ? AND deleted_at IS NULL", folder.Name, userID)
	if parentID != nil {
		query = query.Where("parent_id = ?", parentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}
	query.Count(&conflictCount)
	if conflictCount > 0 {
		return nil, errors.New("目标文件夹中已存在同名文件夹")
	}

	// 6. Update the parent_id
	var updatedAt time.Time
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Update parent_id
		if err := tx.Model(&folder).Update("parent_id", parentID).Error; err != nil {
			return err
		}

		// Update updated_at timestamp
		now := time.Now()
		updatedAt = now
		return tx.Model(&folder).Update("updated_at", now).Error
	})

	if err != nil {
		return nil, err
	}

	return &updatedAt, nil
}

// checkCircularDependency checks if moving sourceFolder into targetParent would create a cycle
// Returns error if moving would create a circular reference
func checkCircularDependency(db *gorm.DB, userID uuid.UUID, sourceFolderID, targetParentID *uuid.UUID) error {
	currentID := targetParentID

	// Traverse up the parent chain
	for currentID != nil {
		if *currentID == *sourceFolderID {
			return ErrFolderMoveCycle
		}

		// Get the parent folder
		var parent models.Folder
		result := db.Where("id = ? AND owner_user_id = ? AND deleted_at IS NULL", currentID, userID).First(&parent)
		if result.Error != nil {
			// If parent doesn't exist, we've reached an invalid state, but not a cycle
			break
		}

		currentID = parent.ParentID
	}

	return nil
}

func UpdateDocumentPublicAccess(userID uuid.UUID, documentID uuid.UUID, publicAccess string) error {
	normalized := normalizePublicAccess(publicAccess)
	if normalized == "" {
		return ErrPublicAccessInvalid
	}

	document, _, err := acl.AuthorizeDocumentAction(database.DB, userID, documentID, acl.ActionOwnerOnly)
	if err != nil {
		return ErrDocumentNotFoundOrUnauthorized
	}

	if normalizePublicAccess(document.PublicAccess) == normalized {
		return nil
	}

	return database.DB.Model(document).Updates(map[string]any{
		"public_access": normalized,
		"updated_by":    userID,
	}).Error
}

// Helper function for Go versions without built-in max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GetCreatorInfo fetches creator information for a user
func GetCreatorInfo(userID uuid.UUID) (*CreatorInfo, error) {
	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}

	return &CreatorInfo{
		ID:          user.ID,
		DisplayName: user.DisplayName,
	}, nil
}

// GetFolderAncestors retrieves the ancestor path of a folder using an iterative approach.
func GetFolderAncestors(userID uuid.UUID, folderID uuid.UUID) ([]AncestorItem, error) {
	var ancestors []AncestorItem
	currentID := &folderID

	// Loop up to 100 times to prevent infinite loops in case of data cycles
	for i := 0; i < 100 && currentID != nil; i++ {
		var folder models.Folder
		result := database.DB.Where("id = ? AND owner_user_id = ? AND deleted_at IS NULL", currentID, userID).First(&folder)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				// We've hit a folder that doesn't exist or a parent that is null.
				// This is a normal termination condition.
				break
			}
			// This is a real database error.
			return nil, result.Error
		}

		ancestors = append(ancestors, AncestorItem{ID: folder.ID, Name: folder.Name})
		currentID = folder.ParentID
	}

	// The loop gets the path from child -> root, so we must reverse it.
	for i, j := 0, len(ancestors)-1; i < j; i, j = i+1, j-1 {
		ancestors[i], ancestors[j] = ancestors[j], ancestors[i]
	}

	return ancestors, nil
}

// TrashListResponse defines the structure for the trashed items list API response
type TrashListResponse struct {
	Items   []TrashItem `json:"items"`
	HasMore bool        `json:"hasMore"`
	Total   int64       `json:"total"`
}

// TrashItem defines the structure for a single item in the trash list
type TrashItem struct {
	ID        uuid.UUID      `json:"id"`
	Type      string         `json:"type"`
	Name      string         `json:"name"`
	DeletedAt gorm.DeletedAt `json:"deletedAt"`
}

// GetTrashedFiles retrieves a list of soft-deleted files for a given user
func GetTrashedFiles(userID uuid.UUID, limit, offset int, sortBy, order string) (*TrashListResponse, error) {
	db := database.DB

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	if sortBy == "" {
		sortBy = "deleted_at"
	}
	if order != "asc" && order != "desc" {
		order = "desc"
	}
	if sortBy != "deleted_at" && sortBy != "name" {
		sortBy = "deleted_at"
	}

	// 1. Find all soft-deleted folder IDs, correctly using Unscoped
	var deletedFolderIDs []uuid.UUID
	db.Unscoped().Model(&models.Folder{}).
		Where("owner_user_id = ? AND deleted_at IS NOT NULL", userID).
		Pluck("id", &deletedFolderIDs)

	// 2. Find top-level soft-deleted folders
	// A folder is top-level if its parent_id is NULL or its parent is NOT in the set of deleted folders
	var topLevelFolders []models.Folder
	queryFolders := db.Unscoped().Model(&models.Folder{}).
		Where("owner_user_id = ? AND deleted_at IS NOT NULL", userID)

	// We only want to see items whose parent is not also in the trash.
	// If no folders are in the trash, this condition is not needed.
	if len(deletedFolderIDs) > 0 {
		queryFolders = queryFolders.Where("parent_id IS NULL OR parent_id NOT IN ?", deletedFolderIDs)
	}
	queryFolders.Find(&topLevelFolders)

	// 3. Find top-level soft-deleted documents
	// A document is top-level if its folder_id is NULL or its folder is NOT in the set of deleted folders
	var topLevelDocuments []models.Document
	queryDocuments := db.Unscoped().Model(&models.Document{}).
		Where("owner_user_id = ? AND deleted_at IS NOT NULL", userID)

	if len(deletedFolderIDs) > 0 {
		queryDocuments = queryDocuments.Where("folder_id IS NULL OR folder_id NOT IN ?", deletedFolderIDs)
	}
	queryDocuments.Find(&topLevelDocuments)

	// 4. Combine and convert to TrashItem DTO
	var items []TrashItem
	for _, f := range topLevelFolders {
		items = append(items, TrashItem{
			ID:        f.ID,
			Type:      "folder",
			Name:      f.Name,
			DeletedAt: f.DeletedAt,
		})
	}
	for _, m := range topLevelDocuments {
		items = append(items, TrashItem{
			ID:        m.ID,
			Type:      "document",
			Name:      m.Title,
			DeletedAt: m.DeletedAt,
		})
	}

	// 5. Sort the combined list (in-memory sort)
	slices.SortStableFunc(items, func(a, b TrashItem) int {
		var comparison int
		switch sortBy {
		case "name":
			comparison = strings.Compare(a.Name, b.Name)
			if comparison == 0 {
				comparison = a.DeletedAt.Time.Compare(b.DeletedAt.Time)
			}
		default:
			comparison = a.DeletedAt.Time.Compare(b.DeletedAt.Time)
			if comparison == 0 {
				comparison = strings.Compare(a.Name, b.Name)
			}
		}

		if order == "desc" {
			comparison = -comparison
		}

		if comparison == 0 {
			return strings.Compare(a.ID.String(), b.ID.String())
		}

		return comparison
	})

	total := int64(len(items))
	// Manual pagination
	start := offset
	end := offset + limit
	if start > len(items) {
		start = len(items)
	}
	if end > len(items) {
		end = len(items)
	}
	paginatedItems := items[start:end]

	return &TrashListResponse{
		Items:   paginatedItems,
		HasMore: int64(offset+len(paginatedItems)) < total,
		Total:   total,
	}, nil
}

// RestoreTrashResponse defines the structure for the restore items API response
type RestoreTrashResponse struct {
	Success       bool         `json:"success"`
	Message       string       `json:"message"`
	RestoredCount int          `json:"restoredCount"`
	FailedItems   []FailedItem `json:"failedItems"`
}

type ItemToRestore struct {
	ID   uuid.UUID `json:"id"`
	Type string    `json:"type"`
}

// RestoreTrashedItems restores a list of soft-deleted items
func RestoreTrashedItems(userID uuid.UUID, itemsToRestore []ItemToRestore) (*RestoreTrashResponse, error) {
	var restoredCount int
	var failedItems []FailedItem

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range itemsToRestore {
			if item.Type == "folder" {
				var folder models.Folder
				// Find the folder, including soft-deleted ones
				if err := tx.Unscoped().Where("id = ? AND owner_user_id = ?", item.ID, userID).First(&folder).Error; err != nil {
					failedItems = append(failedItems, FailedItem{ID: item.ID, Type: item.Type, Reason: "项目不存在。"})
					continue
				}

				// Check for naming conflict in parent directory
				var conflictCount int64
				tx.Model(&models.Folder{}).Where("parent_id = ? AND name = ? AND owner_user_id = ? AND deleted_at IS NULL", folder.ParentID, folder.Name, userID).Count(&conflictCount)
				if conflictCount > 0 {
					failedItems = append(failedItems, FailedItem{ID: item.ID, Type: item.Type, Reason: "恢复失败，目标位置已存在同名文件夹。"})
					continue
				}

				// Recursively restore the folder
				if err := restoreFolderRecursive(tx, userID, item.ID); err != nil {
					return err // Rollback transaction on error
				}
				restoredCount++
			} else if item.Type == "document" {
				document, err := acl.CanAccessDocumentOwnerOnlyUnscoped(tx, userID, item.ID)
				if err != nil {
					failedItems = append(failedItems, FailedItem{ID: item.ID, Type: item.Type, Reason: "项目不存在。"})
					continue
				}

				// Check for naming conflict
				var conflictCount int64
				tx.Model(&models.Document{}).Where("folder_id = ? AND title = ? AND owner_user_id = ? AND deleted_at IS NULL", document.FolderID, document.Title, document.OwnerUserID).Count(&conflictCount)
				if conflictCount > 0 {
					failedItems = append(failedItems, FailedItem{ID: item.ID, Type: item.Type, Reason: "恢复失败，目标位置已存在同名文档。"})
					continue
				}

				// Restore the document
				if err := tx.Unscoped().Model(&models.Document{}).Where("id = ?", item.ID).Update("deleted_at", nil).Error; err != nil {
					return err
				}
				if err := content.RestoreContentByDocumentID(tx, userID, item.ID); err != nil {
					return err
				}
				restoredCount++
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &RestoreTrashResponse{
		Success:       true,
		Message:       "恢复操作完成。",
		RestoredCount: restoredCount,
		FailedItems:   failedItems,
	}, nil
}

// restoreFolderRecursive recursively restores a folder and its contents
func restoreFolderRecursive(tx *gorm.DB, userID uuid.UUID, folderID uuid.UUID) error {
	// Restore the folder itself
	if err := tx.Unscoped().Model(&models.Folder{}).Where("id = ? AND owner_user_id = ?", folderID, userID).Update("deleted_at", nil).Error; err != nil {
		return err
	}

	// Restore all documents in this folder
	if err := tx.Unscoped().Model(&models.Document{}).Where("folder_id = ? AND owner_user_id = ?", folderID, userID).Update("deleted_at", nil).Error; err != nil {
		return err
	}
	var documents []models.Document
	if err := tx.Unscoped().Where("folder_id = ? AND owner_user_id = ?", folderID, userID).Find(&documents).Error; err != nil {
		return err
	}
	for _, document := range documents {
		if err := content.RestoreContentByDocumentID(tx, userID, document.ID); err != nil {
			return err
		}
	}

	// Find all soft-deleted child folders that were deleted at the same time or after the parent
	var childFolders []models.Folder
	if err := tx.Unscoped().Where("parent_id = ? AND owner_user_id = ?", folderID, userID).Find(&childFolders).Error; err != nil {
		return err
	}

	for _, child := range childFolders {
		// Recursively restore child folders
		if err := restoreFolderRecursive(tx, userID, child.ID); err != nil {
			return err
		}
	}

	return nil
}

// PermanentDeleteResponse defines the structure for the permanent delete API response
type PermanentDeleteResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	DeletedCount int64  `json:"deletedCount"`
}

// PermanentDeleteItems permanently deletes items from the trash
func PermanentDeleteItems(userID uuid.UUID, itemsToDelete []ItemToRestore) (*PermanentDeleteResponse, error) {
	var deletedCount int64 = 0

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// If no items are specified, empty the entire trash for the user
		if len(itemsToDelete) == 0 {
			var folders []models.Folder
			if err := tx.Unscoped().Where("owner_user_id = ? AND deleted_at IS NOT NULL", userID).Find(&folders).Error; err != nil {
				return err
			}
			for _, f := range folders {
				if err := permanentDeleteFolderRecursive(tx, userID, f.ID); err != nil {
					return err
				}
				deletedCount++
			}

			var documents []models.Document
			if err := tx.Unscoped().Where("owner_user_id = ? AND deleted_at IS NOT NULL", userID).Find(&documents).Error; err != nil {
				return err
			}
			for _, document := range documents {
				if err := content.PermanentDeleteContentByDocumentID(tx, userID, document.ID); err != nil {
					return err
				}
			}
			result := tx.Unscoped().
				Where("owner_user_id = ? AND deleted_at IS NOT NULL", userID).
				Delete(&models.Document{})
			if result.Error != nil {
				return result.Error
			}
			deletedCount += result.RowsAffected
			return nil
		}

		// If specific items are provided for deletion
		for _, item := range itemsToDelete {
			if item.Type == "folder" {
				if err := permanentDeleteFolderRecursive(tx, userID, item.ID); err != nil {
					// We might want to collect errors instead of failing the whole transaction
					return err
				}
				deletedCount++ // This only counts the top-level folder
			} else if item.Type == "document" {
				if err := content.PermanentDeleteContentByDocumentID(tx, userID, item.ID); err != nil {
					return err
				}
				if _, err := acl.CanAccessDocumentOwnerOnlyUnscoped(tx, userID, item.ID); err != nil {
					continue
				}
				result := tx.Unscoped().Where("id = ?", item.ID).Delete(&models.Document{})
				if result.Error != nil {
					return result.Error
				}
				deletedCount += result.RowsAffected
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &PermanentDeleteResponse{
		Success:      true,
		Message:      "永久删除操作完成。",
		DeletedCount: deletedCount,
	}, nil
}

// permanentDeleteFolderRecursive permanently deletes a folder and all its children
func permanentDeleteFolderRecursive(tx *gorm.DB, userID uuid.UUID, folderID uuid.UUID) error {
	// Find all child folders (including soft-deleted ones)
	var childFolders []models.Folder
	if err := tx.Unscoped().Where("parent_id = ? AND owner_user_id = ?", folderID, userID).Find(&childFolders).Error; err != nil {
		return err
	}

	// Recursively delete child folders
	for _, child := range childFolders {
		if err := permanentDeleteFolderRecursive(tx, userID, child.ID); err != nil {
			return err
		}
	}

	// Permanently delete all documents in this folder
	var documents []models.Document
	if err := tx.Unscoped().Where("folder_id = ? AND owner_user_id = ?", folderID, userID).Find(&documents).Error; err != nil {
		return err
	}
	for _, document := range documents {
		if err := content.PermanentDeleteContentByDocumentID(tx, userID, document.ID); err != nil {
			return err
		}
	}
	if err := tx.Unscoped().Where("folder_id = ? AND owner_user_id = ?", folderID, userID).Delete(&models.Document{}).Error; err != nil {
		return err
	}

	// Permanently delete the folder itself
	if err := tx.Unscoped().Where("id = ? AND owner_user_id = ?", folderID, userID).Delete(&models.Folder{}).Error; err != nil {
		return err
	}

	return nil
}

// BatchMoveFiles moves multiple files and folders to a new destination.
func BatchMoveFiles(userID uuid.UUID, itemsToMove []ItemToMove, destFolderID *uuid.UUID) (*BatchMoveResponse, error) {
	var movedCount int
	var failedItems []FailedItem

	// Use a single transaction so authorization + validation + write stay consistent.
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := validateBatchMoveDestination(tx, userID, destFolderID); err != nil {
			return err
		}

		foldersToMove, documentsToMove, normalizeFailedItems := normalizeBatchMoveItems(itemsToMove)
		failedItems = append(failedItems, normalizeFailedItems...)

		folderMap, excludedFolders, failedFolderItems, err := collectAuthorizedFoldersForMove(tx, userID, foldersToMove)
		if err != nil {
			return err
		}
		failedItems = append(failedItems, failedFolderItems...)

		documentMap, excludedDocuments, failedDocumentItems := collectAuthorizedDocumentsForMove(tx, userID, documentsToMove)
		failedItems = append(failedItems, failedDocumentItems...)

		// A. Circular dependency and self-move checks
		if destFolderID != nil {
			for _, folderID := range foldersToMove {
				if _, excluded := excludedFolders[folderID]; excluded {
					continue
				}
				if folderID == *destFolderID {
					failedItems = append(failedItems, FailedItem{ID: folderID, Type: "folder", Reason: "不能将文件夹移动到其自身内部"})
					excludedFolders[folderID] = struct{}{}
					continue
				}
				if err := checkCircularDependency(tx, userID, &folderID, destFolderID); err != nil {
					failedItems = append(failedItems, FailedItem{ID: folderID, Type: "folder", Reason: err.Error()})
					excludedFolders[folderID] = struct{}{}
				}
			}
		}

		// B. Naming conflict checks
		// Get existing names in destination
		var existingFolders []models.Folder
		var existingDocuments []models.Document
		destQueryFolder := tx.Model(&models.Folder{}).Where("owner_user_id = ? AND deleted_at IS NULL", userID)
		destQueryDocument := tx.Model(&models.Document{}).Where("owner_user_id = ? AND deleted_at IS NULL", userID)

		if destFolderID != nil {
			destQueryFolder = destQueryFolder.Where("parent_id = ?", destFolderID)
			destQueryDocument = destQueryDocument.Where("folder_id = ?", destFolderID)
		} else {
			destQueryFolder = destQueryFolder.Where("parent_id IS NULL")
			destQueryDocument = destQueryDocument.Where("folder_id IS NULL")
		}
		if err := destQueryFolder.Find(&existingFolders).Error; err != nil {
			return err
		}
		if err := destQueryDocument.Find(&existingDocuments).Error; err != nil {
			return err
		}

		existingNames := make(map[string]bool)
		for _, f := range existingFolders {
			existingNames["folder_"+f.Name] = true
		}
		for _, m := range existingDocuments {
			existingNames["document_"+m.Title] = true
		}

		// Check for conflicts
		if len(foldersToMove) > 0 {
			for _, folderID := range foldersToMove {
				// Don't check items that already failed validation
				if _, excluded := excludedFolders[folderID]; excluded {
					continue
				}
				f := folderMap[folderID]
				if existingNames["folder_"+f.Name] {
					failedItems = append(failedItems, FailedItem{ID: f.ID, Type: "folder", Reason: "目标位置已存在同名文件夹"})
					excludedFolders[f.ID] = struct{}{}
				} else {
					// Add to map to check for self-conflicts within the moved items
					existingNames["folder_"+f.Name] = true
				}
			}
		}
		if len(documentsToMove) > 0 {
			for _, documentID := range documentsToMove {
				// Don't check items that already failed validation
				if _, excluded := excludedDocuments[documentID]; excluded {
					continue
				}
				m := documentMap[documentID]
				if existingNames["document_"+m.Title] {
					failedItems = append(failedItems, FailedItem{ID: m.ID, Type: "document", Reason: "目标位置已存在同名文档"})
					excludedDocuments[m.ID] = struct{}{}
				} else {
					// Add to map to check for self-conflicts within the moved items
					existingNames["document_"+m.Title] = true
				}
			}
		}

		// --- 2. EXECUTION PHASE ---

		finalFoldersToMove := []uuid.UUID{}
		for _, id := range foldersToMove {
			if _, excluded := excludedFolders[id]; !excluded {
				finalFoldersToMove = append(finalFoldersToMove, id)
			}
		}
		finalDocumentsToMove := []uuid.UUID{}
		for _, id := range documentsToMove {
			if _, excluded := excludedDocuments[id]; !excluded {
				finalDocumentsToMove = append(finalDocumentsToMove, id)
			}
		}

		now := time.Now()
		if len(finalFoldersToMove) > 0 {
			result := tx.Model(&models.Folder{}).
				Where("id IN ? AND owner_user_id = ? AND deleted_at IS NULL", finalFoldersToMove, userID).
				Updates(map[string]interface{}{"parent_id": destFolderID, "updated_at": now})
			if result.Error != nil {
				// If this fails, it's a serious DB error, rollback everything
				return result.Error
			}
			if int(result.RowsAffected) != len(finalFoldersToMove) {
				return errors.New("部分文件夹状态已变更，请重试")
			}
			movedCount += int(result.RowsAffected)
		}
		if len(finalDocumentsToMove) > 0 {
			result := tx.Model(&models.Document{}).
				Where("id IN ? AND deleted_at IS NULL", finalDocumentsToMove).
				Updates(map[string]interface{}{"folder_id": destFolderID, "updated_at": now})
			if result.Error != nil {
				return result.Error
			}
			if int(result.RowsAffected) != len(finalDocumentsToMove) {
				return errors.New("部分文档状态已变更，请重试")
			}
			movedCount += int(result.RowsAffected)
		}

		return nil // Commit transaction
	})

	if err != nil {
		return nil, err // This will be the destination folder validation error or a DB error
	}

	return &BatchMoveResponse{
		Success:     len(failedItems) == 0,
		Message:     fmt.Sprintf("移动操作完成。成功 %d 个，失败 %d 个。", movedCount, len(failedItems)),
		MovedCount:  movedCount,
		FailedItems: failedItems,
	}, nil
}
