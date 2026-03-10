package workspace

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/content"
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
		query := database.DB.Model(&models.Folder{}).Where("user_id = ? AND deleted_at IS NULL", userID)
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
		if err := query.Select("id", "user_id", "folder_id", "title", "excerpt", "created_at", "updated_at", "created_by").Order(sortBy + " " + order).Limit(limit).Offset(offset).Find(&markdowns).Error; err != nil {
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
		HasMore: int64(offset+len(items)) < total,
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

// CreateMarkdown creates a new markdown document with unique title handling
func CreateMarkdown(userID uuid.UUID, title string, contentStr string, folderID *uuid.UUID) (*models.Markdown, error) {
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

	newTitle := strings.TrimSpace(title)
	if newTitle == "" || newTitle == "未命名文档" {
		// Generate a unique title based on date and sequence
		datePrefix := time.Now().Format("060102")
		baseTitleWithDate := "未命名文档" + datePrefix

		var count int64
		query := database.DB.Model(&models.Markdown{}).Where("user_id = ? AND title LIKE ? AND deleted_at IS NULL", userID, baseTitleWithDate+"-%")
		if folderID != nil {
			query = query.Where("folder_id = ?", folderID)
		} else {
			query = query.Where("folder_id IS NULL")
		}
		query.Count(&count)

		newTitle = fmt.Sprintf("%s-%02d", baseTitleWithDate, count+1)
	} else {
		// For custom titles, check for duplicates in the same folder
		var existing models.Markdown
		query := database.DB.Model(&models.Markdown{}).Where("user_id = ? AND title = ? AND deleted_at IS NULL", userID, newTitle)
		if folderID != nil {
			query = query.Where("folder_id = ?", folderID)
		} else {
			query = query.Where("folder_id IS NULL")
		}
		result := query.First(&existing)
		if result.Error == nil {
			// A record was found, so it's a duplicate
			return nil, errors.New("同名文档已存在")
		}
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// A real database error occurred
			return nil, result.Error
		}
	}

	// Generate excerpt from content
	excerpt := generateExcerpt(contentStr)

	// Create the markdown in a transaction
	var markdown *models.Markdown
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Create markdown metadata
		markdown = &models.Markdown{
			ID:        uuid.New(),
			UserID:    userID,
			FolderID:  folderID,
			Title:     newTitle,
			Excerpt:   excerpt,
			CreatedBy: userID,
		}

		if err := tx.Create(markdown).Error; err != nil {
			return err
		}

		// Create initial content (version 1)
		if err := content.CreateInitialContent(tx, markdown.ID, contentStr); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return markdown, nil
}

// BatchDeleteFiles soft deletes multiple files (folders and markdowns) in a single transaction
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
			} else if item.Type == "markdown" {
				// Delete content first
				if err := content.DeleteContentByMarkdownID(tx, userID, item.ID); err != nil {
					failedItems = append(failedItems, FailedItem{
						ID:     item.ID,
						Type:   item.Type,
						Reason: err.Error(),
					})
					continue
				}

				// Soft delete markdown metadata
				result := tx.Where("id = ? AND user_id = ?", item.ID, userID).Delete(&models.Markdown{})
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

// DeleteFile soft deletes a file (folder or markdown)
func DeleteFile(userID uuid.UUID, fileID uuid.UUID, fileType string) error {
	if fileType == "folder" {
		// Start a single transaction for the entire recursive operation
		return database.DB.Transaction(func(tx *gorm.DB) error {
			return deleteFolderRecursive(tx, userID, fileID)
		})
	} else if fileType == "markdown" {
		// Start a transaction to delete markdown and its content
		return database.DB.Transaction(func(tx *gorm.DB) error {
			// Delete content first
			if err := content.DeleteContentByMarkdownID(tx, userID, fileID); err != nil {
				return err
			}

			// Soft delete markdown metadata
			result := tx.Where("id = ? AND user_id = ?", fileID, userID).Delete(&models.Markdown{})
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
	if err := tx.Where("parent_id = ? AND user_id = ?", folderID, userID).Find(&childFolders).Error; err != nil {
		return err
	}

	// Recursively delete child folders, passing the transaction down
	for _, child := range childFolders {
		if err := deleteFolderRecursive(tx, userID, child.ID); err != nil {
			return err
		}
	}

	// Find all markdowns in this folder
	var markdowns []models.Markdown
	if err := tx.Where("folder_id = ? AND user_id = ?", folderID, userID).Find(&markdowns).Error; err != nil {
		return err
	}

	// Delete content for each markdown, then soft delete the markdown
	for _, md := range markdowns {
		// Delete content
		if err := content.DeleteContentByMarkdownID(tx, userID, md.ID); err != nil {
			return err
		}
		// Soft delete markdown metadata
		if err := tx.Where("id = ?", md.ID).Delete(&models.Markdown{}).Error; err != nil {
			return err
		}
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

// GetFile retrieves a single file (folder or markdown) by ID
func GetFile(userID uuid.UUID, fileID uuid.UUID, fileType string) (*FileItem, error) {
	if fileType == "folder" {
		var folder models.Folder
		result := database.DB.Where("id = ? AND user_id = ?", fileID, userID).First(&folder)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, errors.New("文件不存在")
			}
			return nil, result.Error
		}

		item := folderToFileItem(folder)
		return &item, nil
	} else if fileType == "markdown" {
		var markdown models.Markdown
		result := database.DB.Where("id = ? AND user_id = ?", fileID, userID).First(&markdown)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, errors.New("文件不存在")
			}
			return nil, result.Error
		}

		item := markdownToFileItem(markdown)
		return &item, nil
	}

	return nil, errors.New("无效的文件类型")
}

// UpdateMarkdownTitle updates the title of a markdown document
func UpdateMarkdownTitle(userID uuid.UUID, markdownID uuid.UUID, title string) error {
	trimmedTitle := strings.TrimSpace(title)

	// Validate title length
	if len(trimmedTitle) == 0 {
		return errors.New("文档标题不能为空")
	}
	if len(trimmedTitle) > 255 {
		return errors.New("文档标题不能超过 255 个字符")
	}

	// Verify the markdown exists and belongs to the user
	var markdown models.Markdown
	result := database.DB.Where("id = ? AND user_id = ?", markdownID, userID).First(&markdown)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return errors.New("文档不存在")
		}
		return result.Error
	}

	// Same title is treated as a no-op.
	if markdown.Title == trimmedTitle {
		return nil
	}

	// Check duplicate title in the same folder.
	var conflictCount int64
	query := database.DB.Model(&models.Markdown{}).
		Where("user_id = ? AND id <> ? AND title = ? AND deleted_at IS NULL", userID, markdownID, trimmedTitle)
	if markdown.FolderID != nil {
		query = query.Where("folder_id = ?", markdown.FolderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}
	if err := query.Count(&conflictCount).Error; err != nil {
		return err
	}
	if conflictCount > 0 {
		return errors.New("同名文档已存在")
	}

	// Update the title
	return database.DB.Model(&markdown).Update("title", trimmedTitle).Error
}

// UpdateFolderName updates the name of a folder
func UpdateFolderName(userID uuid.UUID, folderID uuid.UUID, name string) error {
	trimmedName := strings.TrimSpace(name)

	// Validate name length
	if len(trimmedName) == 0 {
		return errors.New("文件夹名称不能为空")
	}
	if len(trimmedName) > 255 {
		return errors.New("文件夹名称不能超过 255 个字符")
	}

	lowerName := strings.ToLower(trimmedName)
	for _, reserved := range ReservedFolderNames {
		if lowerName == strings.ToLower(reserved) {
			return errors.New("不能使用系统保留的文件夹名称")
		}
	}

	// Verify the folder exists and belongs to the user
	var folder models.Folder
	result := database.DB.Where("id = ? AND user_id = ?", folderID, userID).First(&folder)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return errors.New("文件夹不存在")
		}
		return result.Error
	}

	// Same name is treated as a no-op.
	if folder.Name == trimmedName {
		return nil
	}

	var conflictCount int64
	query := database.DB.Model(&models.Folder{}).
		Where("user_id = ? AND id <> ? AND name = ? AND deleted_at IS NULL", userID, folderID, trimmedName)
	if folder.ParentID != nil {
		query = query.Where("parent_id = ?", folder.ParentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}
	if err := query.Count(&conflictCount).Error; err != nil {
		return err
	}
	if conflictCount > 0 {
		return errors.New("同名文件夹已存在")
	}

	// Update the name
	return database.DB.Model(&folder).Update("name", trimmedName).Error
}

// MoveMarkdown moves a markdown document to a different folder (or root)
func MoveMarkdown(userID uuid.UUID, markdownID uuid.UUID, folderID *uuid.UUID) (*time.Time, error) {
	// 1. Verify the markdown exists, belongs to the user, and is not deleted
	var markdown models.Markdown
	result := database.DB.Where("id = ? AND user_id = ? AND deleted_at IS NULL", markdownID, userID).First(&markdown)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("文档不存在或已被删除")
		}
		return nil, result.Error
	}

	// 2. Validate target folder if provided
	if folderID != nil {
		var folder models.Folder
		result = database.DB.Where("id = ? AND user_id = ? AND deleted_at IS NULL", folderID, userID).First(&folder)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, errors.New("目标文件夹不存在或已被删除")
			}
			return nil, result.Error
		}
	}

	// 3. Check for naming conflict in the destination
	var conflictCount int64
	query := database.DB.Model(&models.Markdown{}).Where("title = ? AND user_id = ? AND deleted_at IS NULL", markdown.Title, userID)
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
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Update folder_id
		if err := tx.Model(&markdown).Update("folder_id", folderID).Error; err != nil {
			return err
		}

		// Update updated_at timestamp
		now := time.Now()
		updatedAt = now
		return tx.Model(&markdown).Update("updated_at", now).Error
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
	result := database.DB.Where("id = ? AND user_id = ? AND deleted_at IS NULL", folderID, userID).First(&folder)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("文件夹不存在或已被删除")
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
		result = database.DB.Where("id = ? AND user_id = ? AND deleted_at IS NULL", parentID, userID).First(&parentFolder)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, errors.New("目标父文件夹不存在或已被删除")
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
	query := database.DB.Model(&models.Folder{}).Where("name = ? AND user_id = ? AND deleted_at IS NULL", folder.Name, userID)
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
			return errors.New("不能将文件夹移动到其子文件夹下")
		}

		// Get the parent folder
		var parent models.Folder
		result := db.Where("id = ? AND user_id = ? AND deleted_at IS NULL", currentID, userID).First(&parent)
		if result.Error != nil {
			// If parent doesn't exist, we've reached an invalid state, but not a cycle
			break
		}

		currentID = parent.ParentID
	}

	return nil
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
				if err := content.RestoreContentByMarkdownID(tx, userID, item.ID); err != nil {
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
	var markdowns []models.Markdown
	if err := tx.Unscoped().Where("folder_id = ? AND user_id = ?", folderID, userID).Find(&markdowns).Error; err != nil {
		return err
	}
	for _, markdown := range markdowns {
		if err := content.RestoreContentByMarkdownID(tx, userID, markdown.ID); err != nil {
			return err
		}
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
			tx.Unscoped().Where("user_id = ? AND deleted_at IS NOT NULL", userID).Find(&folders)
			for _, f := range folders {
				if err := permanentDeleteFolderRecursive(tx, userID, f.ID); err != nil {
					return err
				}
				deletedCount++
			}

			var markdowns []models.Markdown
			tx.Unscoped().Where("user_id = ? AND deleted_at IS NOT NULL", userID).Find(&markdowns)
			for _, markdown := range markdowns {
				if err := content.PermanentDeleteContentByMarkdownID(tx, userID, markdown.ID); err != nil {
					return err
				}
			}
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
				if err := content.PermanentDeleteContentByMarkdownID(tx, userID, item.ID); err != nil {
					return err
				}
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
	var markdowns []models.Markdown
	if err := tx.Unscoped().Where("folder_id = ? AND user_id = ?", folderID, userID).Find(&markdowns).Error; err != nil {
		return err
	}
	for _, markdown := range markdowns {
		if err := content.PermanentDeleteContentByMarkdownID(tx, userID, markdown.ID); err != nil {
			return err
		}
	}
	if err := tx.Unscoped().Where("folder_id = ? AND user_id = ?", folderID, userID).Delete(&models.Markdown{}).Error; err != nil {
		return err
	}

	// Permanently delete the folder itself
	if err := tx.Unscoped().Where("id = ? AND user_id = ?", folderID, userID).Delete(&models.Folder{}).Error; err != nil {
		return err
	}

	return nil
}

// BatchMoveFiles moves multiple files and folders to a new destination.
func BatchMoveFiles(userID uuid.UUID, itemsToMove []ItemToMove, destFolderID *uuid.UUID) (*BatchMoveResponse, error) {
	var movedCount int
	var failedItems []FailedItem

	// Use a transaction for the entire operation
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// --- 1. PRE-VALIDATION PHASE ---

		// Validate destination folder exists and belongs to the user
		if destFolderID != nil {
			var destFolder models.Folder
			if err := tx.Where("id = ? AND user_id = ? AND deleted_at IS NULL", destFolderID, userID).First(&destFolder).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errors.New("目标文件夹不存在或已被删除")
				}
				return err
			}
		}

		// Separate items by type and deduplicate IDs.
		var foldersToMove []uuid.UUID
		var markdownsToMove []uuid.UUID
		seenFolders := make(map[uuid.UUID]bool)
		seenMarkdowns := make(map[uuid.UUID]bool)
		for _, item := range itemsToMove {
			if item.Type == "folder" {
				if !seenFolders[item.ID] {
					seenFolders[item.ID] = true
					foldersToMove = append(foldersToMove, item.ID)
				}
			} else if item.Type == "markdown" {
				if !seenMarkdowns[item.ID] {
					seenMarkdowns[item.ID] = true
					markdownsToMove = append(markdownsToMove, item.ID)
				}
			} else {
				failedItems = append(failedItems, FailedItem{ID: item.ID, Type: item.Type, Reason: "无效的文件类型"})
			}
		}

		// Items that fail validation will be added to this map.
		itemsToExclude := make(map[uuid.UUID]bool)
		folderMap := make(map[uuid.UUID]models.Folder)
		markdownMap := make(map[uuid.UUID]models.Markdown)

		// Validate all source folders belong to the current user and are not deleted.
		if len(foldersToMove) > 0 {
			var folders []models.Folder
			if err := tx.Where("id IN ? AND user_id = ? AND deleted_at IS NULL", foldersToMove, userID).Find(&folders).Error; err != nil {
				return err
			}
			for _, f := range folders {
				folderMap[f.ID] = f
			}
			for _, folderID := range foldersToMove {
				if _, ok := folderMap[folderID]; !ok {
					failedItems = append(failedItems, FailedItem{ID: folderID, Type: "folder", Reason: "文件夹不存在或无权操作"})
					itemsToExclude[folderID] = true
				}
			}
		}

		// Validate all source markdowns belong to the current user and are not deleted.
		if len(markdownsToMove) > 0 {
			var markdowns []models.Markdown
			if err := tx.Where("id IN ? AND user_id = ? AND deleted_at IS NULL", markdownsToMove, userID).Find(&markdowns).Error; err != nil {
				return err
			}
			for _, m := range markdowns {
				markdownMap[m.ID] = m
			}
			for _, markdownID := range markdownsToMove {
				if _, ok := markdownMap[markdownID]; !ok {
					failedItems = append(failedItems, FailedItem{ID: markdownID, Type: "markdown", Reason: "文档不存在或无权操作"})
					itemsToExclude[markdownID] = true
				}
			}
		}

		// A. Circular dependency and self-move checks
		if destFolderID != nil {
			for _, folderID := range foldersToMove {
				if itemsToExclude[folderID] {
					continue
				}
				if folderID == *destFolderID {
					failedItems = append(failedItems, FailedItem{ID: folderID, Type: "folder", Reason: "不能将文件夹移动到其自身内部"})
					itemsToExclude[folderID] = true
					continue
				}
				if err := checkCircularDependency(tx, userID, &folderID, destFolderID); err != nil {
					failedItems = append(failedItems, FailedItem{ID: folderID, Type: "folder", Reason: err.Error()})
					itemsToExclude[folderID] = true
				}
			}
		}

		// B. Naming conflict checks
		// Get existing names in destination
		var existingFolders []models.Folder
		var existingMarkdowns []models.Markdown
		destQueryFolder := tx.Model(&models.Folder{}).Where("user_id = ? AND deleted_at IS NULL", userID)
		destQueryMarkdown := tx.Model(&models.Markdown{}).Where("user_id = ? AND deleted_at IS NULL", userID)

		if destFolderID != nil {
			destQueryFolder = destQueryFolder.Where("parent_id = ?", destFolderID)
			destQueryMarkdown = destQueryMarkdown.Where("folder_id = ?", destFolderID)
		} else {
			destQueryFolder = destQueryFolder.Where("parent_id IS NULL")
			destQueryMarkdown = destQueryMarkdown.Where("folder_id IS NULL")
		}
		if err := destQueryFolder.Find(&existingFolders).Error; err != nil {
			return err
		}
		if err := destQueryMarkdown.Find(&existingMarkdowns).Error; err != nil {
			return err
		}

		existingNames := make(map[string]bool)
		for _, f := range existingFolders {
			existingNames["folder_"+f.Name] = true
		}
		for _, m := range existingMarkdowns {
			existingNames["markdown_"+m.Title] = true
		}

		// Check for conflicts
		if len(foldersToMove) > 0 {
			for _, folderID := range foldersToMove {
				// Don't check items that already failed validation
				if itemsToExclude[folderID] {
					continue
				}
				f := folderMap[folderID]
				if existingNames["folder_"+f.Name] {
					failedItems = append(failedItems, FailedItem{ID: f.ID, Type: "folder", Reason: "目标位置已存在同名文件夹"})
					itemsToExclude[f.ID] = true
				} else {
					// Add to map to check for self-conflicts within the moved items
					existingNames["folder_"+f.Name] = true
				}
			}
		}
		if len(markdownsToMove) > 0 {
			for _, markdownID := range markdownsToMove {
				// Don't check items that already failed validation
				if itemsToExclude[markdownID] {
					continue
				}
				m := markdownMap[markdownID]
				if existingNames["markdown_"+m.Title] {
					failedItems = append(failedItems, FailedItem{ID: m.ID, Type: "markdown", Reason: "目标位置已存在同名文档"})
					itemsToExclude[m.ID] = true
				} else {
					// Add to map to check for self-conflicts within the moved items
					existingNames["markdown_"+m.Title] = true
				}
			}
		}

		// --- 2. EXECUTION PHASE ---

		finalFoldersToMove := []uuid.UUID{}
		for _, id := range foldersToMove {
			if !itemsToExclude[id] {
				finalFoldersToMove = append(finalFoldersToMove, id)
			}
		}
		finalMarkdownsToMove := []uuid.UUID{}
		for _, id := range markdownsToMove {
			if !itemsToExclude[id] {
				finalMarkdownsToMove = append(finalMarkdownsToMove, id)
			}
		}

		now := time.Now()
		if len(finalFoldersToMove) > 0 {
			result := tx.Model(&models.Folder{}).
				Where("id IN ? AND user_id = ? AND deleted_at IS NULL", finalFoldersToMove, userID).
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
		if len(finalMarkdownsToMove) > 0 {
			result := tx.Model(&models.Markdown{}).
				Where("id IN ? AND user_id = ? AND deleted_at IS NULL", finalMarkdownsToMove, userID).
				Updates(map[string]interface{}{"folder_id": destFolderID, "updated_at": now})
			if result.Error != nil {
				return result.Error
			}
			if int(result.RowsAffected) != len(finalMarkdownsToMove) {
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
