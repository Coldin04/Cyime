package content

import (
	"errors"
	"time"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetContentResult represents the result of getting a document's content
type GetContentResult struct {
	ID         uuid.UUID `json:"id"`
	MarkdownID uuid.UUID `json:"markdownId"`
	Version    int       `json:"version"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"createdAt"`
}

// GetContent retrieves the latest content of a markdown document
func GetContent(userID uuid.UUID, markdownID uuid.UUID) (*GetContentResult, error) {
	// First verify the markdown belongs to the user
	var markdown models.Markdown
	result := database.DB.Where("id = ? AND user_id = ?", markdownID, userID).First(&markdown)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("文档不存在或无权访问")
		}
		return nil, result.Error
	}

	// Get the latest version of the content
	var content models.MarkdownContent
	result = database.DB.Where("markdown_id = ?", markdownID).
		Order("version DESC").
		First(&content)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("文档内容不存在")
		}
		return nil, result.Error
	}

	return &GetContentResult{
		ID:         content.ID,
		MarkdownID: content.MarkdownID,
		Version:    content.Version,
		Content:    content.Content,
		CreatedAt:  content.CreatedAt,
	}, nil
}

// GetContentByVersion retrieves a specific version of a document's content
func GetContentByVersion(userID uuid.UUID, markdownID uuid.UUID, version int) (*GetContentResult, error) {
	// First verify the markdown belongs to the user
	var markdown models.Markdown
	result := database.DB.Where("id = ? AND user_id = ?", markdownID, userID).First(&markdown)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("文档不存在或无权访问")
		}
		return nil, result.Error
	}

	// Get the specific version of the content
	var content models.MarkdownContent
	result = database.DB.Where("markdown_id = ? AND version = ?", markdownID, version).
		First(&content)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("指定版本的内容不存在")
		}
		return nil, result.Error
	}

	return &GetContentResult{
		ID:         content.ID,
		MarkdownID: content.MarkdownID,
		Version:    content.Version,
		Content:    content.Content,
		CreatedAt:  content.CreatedAt,
	}, nil
}

// VersionInfo represents information about a document version
type VersionInfo struct {
	ID        uuid.UUID `json:"id"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
}

// GetVersions retrieves all versions of a document
func GetVersions(userID uuid.UUID, markdownID uuid.UUID) ([]VersionInfo, error) {
	// First verify the markdown belongs to the user
	var markdown models.Markdown
	result := database.DB.Where("id = ? AND user_id = ?", markdownID, userID).First(&markdown)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("文档不存在或无权访问")
		}
		return nil, result.Error
	}

	// Get all versions
	var contents []models.MarkdownContent
	result = database.DB.Where("markdown_id = ?", markdownID).
		Order("version DESC").
		Select("id", "version", "created_at").
		Find(&contents)
	if result.Error != nil {
		return nil, result.Error
	}

	versions := make([]VersionInfo, len(contents))
	for i, c := range contents {
		versions[i] = VersionInfo{
			ID:        c.ID,
			Version:   c.Version,
			CreatedAt: c.CreatedAt,
		}
	}

	return versions, nil
}

// UpdateContentResult represents the result of updating document content
type UpdateContentResult struct {
	Success   bool      `json:"success"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// UpdateContent updates a document's content (creates a new version)
func UpdateContent(userID uuid.UUID, markdownID uuid.UUID, newContent string) (*UpdateContentResult, error) {
	var newVersion int
	var updatedAt time.Time

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Verify the markdown belongs to the user
		var markdown models.Markdown
		result := tx.Where("id = ? AND user_id = ?", markdownID, userID).First(&markdown)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return errors.New("文档不存在或无权访问")
			}
			return result.Error
		}

		// 2. Get the latest version number
		var latestContent models.MarkdownContent
		result = tx.Where("markdown_id = ?", markdownID).
			Order("version DESC").
			First(&latestContent)
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}

		// 3. Create a new version
		newVersion = 1
		if result.Error == nil {
			newVersion = latestContent.Version + 1
		}

		newContentRecord := &models.MarkdownContent{
			ID:         uuid.New(),
			MarkdownID: markdownID,
			Version:    newVersion,
			Content:    newContent,
		}

		if err := tx.Create(newContentRecord).Error; err != nil {
			return err
		}

		// 4. Update the markdown's updated_at timestamp
		if err := tx.Model(&markdown).Update("updated_at", time.Now()).Error; err != nil {
			return err
		}

		updatedAt = time.Now()
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &UpdateContentResult{
		Success:   true,
		Version:   newVersion,
		UpdatedAt: updatedAt,
	}, nil
}

// CreateInitialContent creates the initial content (version 1) for a new document
// This is called when creating a new markdown document
func CreateInitialContent(tx *gorm.DB, markdownID uuid.UUID, content string) error {
	contentRecord := &models.MarkdownContent{
		ID:         uuid.New(),
		MarkdownID: markdownID,
		Version:    1,
		Content:    content,
	}

	return tx.Create(contentRecord).Error
}

// DeleteContentByMarkdownID deletes all content versions for a document
// This is called when deleting a markdown document
func DeleteContentByMarkdownID(tx *gorm.DB, markdownID uuid.UUID) error {
	return tx.Where("markdown_id = ?", markdownID).Delete(&models.MarkdownContent{}).Error
}
