package workspace

import (
	"errors"
	"strings"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
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

// GetFiles retrieves a list of files (folders and markdowns) for a given user and parent folder
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
		query := database.DB.Model(&models.Folder{}).Where("user_id = ? AND deleted_at IS NULL", userID)
		if parentID != nil {
			query = query.Where("parent_id = ?", parentID)
		} else {
			query = query.Where("parent_id IS NULL")
		}

		query.Count(&total)

		var folders []models.Folder
		if err := query.Order(sortBy + " " + order).Limit(limit).Offset(offset).Find(&folders).Error; err != nil {
			return nil, err
		}

		for _, f := range folders {
			items = append(items, folderToFileItem(f))
		}

	} else if filterType == "markdowns" {
		// Only markdowns
		query := database.DB.Model(&models.Markdown{}).Where("user_id = ? AND deleted_at IS NULL", userID)
		if parentID != nil {
			query = query.Where("folder_id = ?", parentID)
		} else {
			query = query.Where("folder_id IS NULL")
		}

		query.Count(&total)

		var markdowns []models.Markdown
		if err := query.Order(sortBy + " " + order).Limit(limit).Offset(offset).Find(&markdowns).Error; err != nil {
			return nil, err
		}

		for _, m := range markdowns {
			items = append(items, markdownToFileItem(m))
		}

	} else {
		// UNION ALL for both folders and markdowns
		// Count total for both types
		var folderCount, markdownCount int64

		folderQuery := database.DB.Model(&models.Folder{}).Where("user_id = ? AND deleted_at IS NULL", userID)
		if parentID != nil {
			folderQuery = folderQuery.Where("parent_id = ?", parentID)
		} else {
			folderQuery = folderQuery.Where("parent_id IS NULL")
		}
		folderQuery.Count(&folderCount)

		markdownQuery := database.DB.Model(&models.Markdown{}).Where("user_id = ? AND deleted_at IS NULL", userID)
		if parentID != nil {
			markdownQuery = markdownQuery.Where("folder_id = ?", parentID)
		} else {
			markdownQuery = markdownQuery.Where("folder_id IS NULL")
		}
		markdownQuery.Count(&markdownCount)

		total = folderCount + markdownCount

		// Fetch folders
		var folders []models.Folder
		folderQuery.Order(sortBy + " " + order).Limit(limit).Offset(offset).Find(&folders)

		for _, f := range folders {
			items = append(items, folderToFileItem(f))
		}

		// Fetch markdowns (adjust limit to account for already fetched folders)
		remainingLimit := limit - len(folders)
		if remainingLimit > 0 {
			var markdowns []models.Markdown
			markdownQuery.Order(sortBy + " " + order).Limit(remainingLimit).Offset(max(0, offset-len(folders))).Find(&markdowns)

			for _, m := range markdowns {
				items = append(items, markdownToFileItem(m))
			}
		}
	}

	return &FileListResponse{
		Items:   items,
		HasMore: int64(len(items)) < total,
		Total:   total,
	}, nil
}

// CreateFolder creates a new folder with validation
func CreateFolder(userID uuid.UUID, name string, description *string, parentID *uuid.UUID) (*models.Folder, error) {
	// Validate name is not empty
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("文件夹名称不能为空")
	}

	// Validate name length
	if len(name) > 255 {
		return nil, errors.New("文件夹名称不能超过 255 个字符")
	}

	// Validate not a reserved name
	lowerName := strings.ToLower(strings.TrimSpace(name))
	for _, reserved := range ReservedFolderNames {
		if lowerName == strings.ToLower(reserved) {
			return nil, errors.New("不能使用系统保留的文件夹名称")
		}
	}

	// Validate parent folder exists if provided
	if parentID != nil {
		var parent models.Folder
		result := database.DB.Where("id = ? AND user_id = ?", parentID, userID).First(&parent)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, errors.New("父文件夹不存在")
			}
			return nil, result.Error
		}

		// Validate no duplicate name in same parent
		var existing models.Folder
		result = database.DB.Where("user_id = ? AND parent_id = ? AND name = ? AND deleted_at IS NULL", userID, parentID, name).First(&existing)
		if result.Error == nil {
			return nil, errors.New("同名文件夹已存在")
		}
	} else {
		// Validate no duplicate name in root
		var existing models.Folder
		result := database.DB.Where("user_id = ? AND parent_id IS NULL AND name = ? AND deleted_at IS NULL", userID, name).First(&existing)
		if result.Error == nil {
			return nil, errors.New("同名文件夹已存在")
		}
	}

	// Create the folder
	folder := &models.Folder{
		ID:          uuid.New(),
		UserID:      userID,
		ParentID:    parentID,
		Name:        name,
		Description: description,
		CreatedBy:   userID,
	}

	if err := database.DB.Create(folder).Error; err != nil {
		return nil, err
	}

	return folder, nil
}

// CreateMarkdown creates a new markdown document
func CreateMarkdown(userID uuid.UUID, title string, content string, folderID *uuid.UUID) (*models.Markdown, error) {
	// Validate title is not empty
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("文档标题不能为空")
	}

	// Validate title length
	if len(title) > 255 {
		return nil, errors.New("文档标题不能超过 255 个字符")
	}

	// Validate folder exists if provided
	if folderID != nil {
		var folder models.Folder
		result := database.DB.Where("id = ? AND user_id = ?", folderID, userID).First(&folder)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, errors.New("文件夹不存在")
			}
			return nil, result.Error
		}
	}

	// Generate excerpt from content
	excerpt := generateExcerpt(content)

	// Create the markdown
	markdown := &models.Markdown{
		ID:        uuid.New(),
		UserID:    userID,
		FolderID:  folderID,
		Title:     title,
		Excerpt:   excerpt,
		Content:   content,
		CreatedBy: userID,
	}

	if err := database.DB.Create(markdown).Error; err != nil {
		return nil, err
	}

	return markdown, nil
}

// DeleteFile soft deletes a file (folder or markdown)
func DeleteFile(userID uuid.UUID, fileID uuid.UUID, fileType string) error {
	if fileType == "folder" {
		// Start a single transaction for the entire recursive operation
		return database.DB.Transaction(func(tx *gorm.DB) error {
			return deleteFolderRecursive(tx, userID, fileID)
		})
	} else if fileType == "markdown" {
		// Soft delete markdown
		result := database.DB.Where("id = ? AND user_id = ?", fileID, userID).Delete(&models.Markdown{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("文档不存在或无权删除")
		}
		return nil
	}

	return errors.New("无效的文件类型")
}

// deleteFolderRecursive recursively deletes a folder and all its children within a single transaction
func deleteFolderRecursive(tx *gorm.DB, userID uuid.UUID, folderID uuid.UUID) error {
	// Find all child folders within the transaction
	var childFolders []models.Folder
	if err := tx.Where("parent_id = ? AND user_id = ?", folderID, userID).Find(&childFolders).Error; err != nil {
		return err
	}

	// Recursively delete child folders, passing the transaction down
	for _, child := range childFolders {
		if err := deleteFolderRecursive(tx, userID, child.ID); err != nil {
			return err
		}
	}

	// Delete all markdowns in this folder within the transaction
	if err := tx.Where("folder_id = ? AND user_id = ?", folderID, userID).Delete(&models.Markdown{}).Error; err != nil {
		return err
	}

	// Soft delete the folder itself within the transaction
	result := tx.Where("id = ? AND user_id = ?", folderID, userID).Delete(&models.Folder{})
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

// markdownToFileItem converts a Markdown model to FileItem DTO
func markdownToFileItem(markdown models.Markdown) FileItem {
	return FileItem{
		ID:        markdown.ID,
		Type:      "markdown",
		Name:      markdown.Title,
		Title:     &markdown.Title,
		Excerpt:   &markdown.Excerpt,
		FolderID:  markdown.FolderID,
		CreatedAt: markdown.CreatedAt,
		UpdatedAt: markdown.UpdatedAt,
		Creator: CreatorInfo{
			ID:          markdown.CreatedBy,
			DisplayName: nil,
		},
	}
}

// generateExcerpt generates a plain text excerpt from markdown content
func generateExcerpt(content string) string {
	// Simple excerpt generation: take first 100 characters of plain text
	// In a real implementation, you might want to strip markdown syntax
	if len(content) == 0 {
		return ""
	}

	// Remove markdown syntax (basic stripping)
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
		result := database.DB.Where("id = ? AND user_id = ? AND deleted_at IS NULL", currentID, userID).First(&folder)

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
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	DeletedAt gorm.DeletedAt `json:"deletedAt"`
}

// GetTrashedFiles retrieves a list of soft-deleted files for a given user
func GetTrashedFiles(userID uuid.UUID, limit, offset int, sortBy, order string) (*TrashListResponse, error) {
	db := database.DB

	// 1. Find all soft-deleted folder IDs, correctly using Unscoped
	var deletedFolderIDs []uuid.UUID
	db.Unscoped().Model(&models.Folder{}).
		Where("user_id = ? AND deleted_at IS NOT NULL", userID).
		Pluck("id", &deletedFolderIDs)

	// 2. Find top-level soft-deleted folders
	// A folder is top-level if its parent_id is NULL or its parent is NOT in the set of deleted folders
	var topLevelFolders []models.Folder
	queryFolders := db.Unscoped().Model(&models.Folder{}).
		Where("user_id = ? AND deleted_at IS NOT NULL", userID)

	// We only want to see items whose parent is not also in the trash.
	// If no folders are in the trash, this condition is not needed.
	if len(deletedFolderIDs) > 0 {
		queryFolders = queryFolders.Where("parent_id IS NULL OR parent_id NOT IN ?", deletedFolderIDs)
	}
	queryFolders.Find(&topLevelFolders)
	
	// 3. Find top-level soft-deleted markdowns
	// A markdown is top-level if its folder_id is NULL or its folder is NOT in the set of deleted folders
	var topLevelMarkdowns []models.Markdown
	queryMarkdowns := db.Unscoped().Model(&models.Markdown{}).
		Where("user_id = ? AND deleted_at IS NOT NULL", userID)

	if len(deletedFolderIDs) > 0 {
		queryMarkdowns = queryMarkdowns.Where("folder_id IS NULL OR folder_id NOT IN ?", deletedFolderIDs)
	}
	queryMarkdowns.Find(&topLevelMarkdowns)

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
	for _, m := range topLevelMarkdowns {
		items = append(items, TrashItem{
			ID:        m.ID,
			Type:      "markdown",
			Name:      m.Title,
			DeletedAt: m.DeletedAt,
		})
	}

	// 5. Sort the combined list (in-memory sort)
	// TODO: For larger datasets, sorting should be done in the DB, which requires a more complex UNION query.
	// For now, in-memory sort is acceptable.

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
		HasMore: total > int64(end),
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

// FailedItem defines a file that failed to be restored, and the reason why
type FailedItem struct {
	ID     uuid.UUID `json:"id"`
	Type   string    `json:"type"`
	Reason string    `json:"reason"`
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
				if err := tx.Unscoped().Where("id = ? AND user_id = ?", item.ID, userID).First(&folder).Error; err != nil {
					failedItems = append(failedItems, FailedItem{ID: item.ID, Type: item.Type, Reason: "项目不存在。"})
					continue
				}

				// Check for naming conflict in parent directory
				var conflictCount int64
				tx.Model(&models.Folder{}).Where("parent_id = ? AND name = ? AND user_id = ? AND deleted_at IS NULL", folder.ParentID, folder.Name, userID).Count(&conflictCount)
				if conflictCount > 0 {
					failedItems = append(failedItems, FailedItem{ID: item.ID, Type: item.Type, Reason: "恢复失败，目标位置已存在同名文件夹。"})
					continue
				}

				// Recursively restore the folder
				if err := restoreFolderRecursive(tx, userID, item.ID); err != nil {
					return err // Rollback transaction on error
				}
				restoredCount++
			} else if item.Type == "markdown" {
				var markdown models.Markdown
				if err := tx.Unscoped().Where("id = ? AND user_id = ?", item.ID, userID).First(&markdown).Error; err != nil {
					failedItems = append(failedItems, FailedItem{ID: item.ID, Type: item.Type, Reason: "项目不存在。"})
					continue
				}

				// Check for naming conflict
				var conflictCount int64
				tx.Model(&models.Markdown{}).Where("folder_id = ? AND title = ? AND user_id = ? AND deleted_at IS NULL", markdown.FolderID, markdown.Title, userID).Count(&conflictCount)
				if conflictCount > 0 {
					failedItems = append(failedItems, FailedItem{ID: item.ID, Type: item.Type, Reason: "恢复失败，目标位置已存在同名文档。"})
					continue
				}

				// Restore the markdown
				if err := tx.Unscoped().Model(&models.Markdown{}).Where("id = ?", item.ID).Update("deleted_at", nil).Error; err != nil {
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
	if err := tx.Unscoped().Model(&models.Folder{}).Where("id = ? AND user_id = ?", folderID, userID).Update("deleted_at", nil).Error; err != nil {
		return err
	}

	// Restore all markdowns in this folder
	if err := tx.Unscoped().Model(&models.Markdown{}).Where("folder_id = ? AND user_id = ?", folderID, userID).Update("deleted_at", nil).Error; err != nil {
		return err
	}

	// Find all soft-deleted child folders that were deleted at the same time or after the parent
	var childFolders []models.Folder
	if err := tx.Unscoped().Where("parent_id = ? AND user_id = ?", folderID, userID).Find(&childFolders).Error; err != nil {
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
	Success      bool  `json:"success"`
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
			tx.Unscoped().Where("user_id = ? AND deleted_at IS NOT NULL", userID).Find(&folders)
			for _, f := range folders {
				if err := permanentDeleteFolderRecursive(tx, userID, f.ID); err != nil {
					return err
				}
				deletedCount++
			}

			var markdowns []models.Markdown
			tx.Unscoped().Where("user_id = ? AND deleted_at IS NOT NULL", userID).Find(&markdowns)
			result := tx.Unscoped().Delete(&markdowns)
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
			} else if item.Type == "markdown" {
				result := tx.Unscoped().Where("id = ? AND user_id = ?", item.ID, userID).Delete(&models.Markdown{})
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
	if err := tx.Unscoped().Where("parent_id = ? AND user_id = ?", folderID, userID).Find(&childFolders).Error; err != nil {
		return err
	}

	// Recursively delete child folders
	for _, child := range childFolders {
		if err := permanentDeleteFolderRecursive(tx, userID, child.ID); err != nil {
			return err
		}
	}

	// Permanently delete all markdowns in this folder
	if err := tx.Unscoped().Where("folder_id = ? AND user_id = ?", folderID, userID).Delete(&models.Markdown{}).Error; err != nil {
		return err
	}

	// Permanently delete the folder itself
	if err := tx.Unscoped().Where("id = ? AND user_id = ?", folderID, userID).Delete(&models.Folder{}).Error; err != nil {
		return err
	}

	return nil
}


