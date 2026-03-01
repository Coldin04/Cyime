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
		// Recursively delete folder and all children
		return deleteFolderRecursive(userID, fileID)
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

// deleteFolderRecursive recursively deletes a folder and all its children
func deleteFolderRecursive(userID uuid.UUID, folderID uuid.UUID) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Find all child folders
		var childFolders []models.Folder
		if err := tx.Where("parent_id = ? AND user_id = ?", folderID, userID).Find(&childFolders).Error; err != nil {
			return err
		}

		// Recursively delete child folders
		for _, child := range childFolders {
			if err := deleteFolderRecursive(userID, child.ID); err != nil {
				return err
			}
		}

		// Delete all markdowns in this folder
		if err := tx.Where("folder_id = ? AND user_id = ?", folderID, userID).Delete(&models.Markdown{}).Error; err != nil {
			return err
		}

		// Soft delete the folder itself
		result := tx.Where("id = ? AND user_id = ?", folderID, userID).Delete(&models.Folder{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("文件夹不存在或无权删除")
		}

		return nil
	})
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
