package content

import (
	"errors"
	"strings"
	"time"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetContentResult represents the current content of a document.
type GetContentResult struct {
	ID              uuid.UUID `json:"id"`
	DocumentID      uuid.UUID `json:"documentId"`
	Content         string    `json:"content"`
	ContentJSON     string    `json:"contentJson"`
	ContentMarkdown string    `json:"contentMarkdown"`
	PlainText       string    `json:"plainText"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// UpdateContentResult represents the result of updating document content.
type UpdateContentResult struct {
	Success   bool      `json:"success"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// GetContent retrieves the current content of a document.
func GetContent(userID uuid.UUID, documentID uuid.UUID) (*GetContentResult, error) {
	var document models.Document
	result := database.DB.Where("id = ? AND owner_user_id = ?", documentID, userID).First(&document)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("文档不存在或无权访问")
		}
		return nil, result.Error
	}

	var content models.DocumentContent
	result = database.DB.Where("document_id = ?", documentID).First(&content)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("文档内容不存在")
		}
		return nil, result.Error
	}

	return &GetContentResult{
		ID:              content.ID,
		DocumentID:      content.DocumentID,
		Content:         currentContentValue(content),
		ContentJSON:     content.ContentJSON,
		ContentMarkdown: content.ContentMarkdown,
		PlainText:       content.PlainText,
		CreatedAt:       content.CreatedAt,
		UpdatedAt:       content.UpdatedAt,
	}, nil
}

// UpdateContent updates the current content of a document in place.
func UpdateContent(userID uuid.UUID, documentID uuid.UUID, newContent string) (*UpdateContentResult, error) {
	var updatedAt time.Time
	plainText := toPlainText(newContent)
	excerpt := buildExcerpt(plainText)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var document models.Document
		result := tx.Where("id = ? AND owner_user_id = ?", documentID, userID).First(&document)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return errors.New("文档不存在或无权访问")
			}
			return result.Error
		}

		now := time.Now()
		result = tx.Model(&models.DocumentContent{}).
			Where("document_id = ?", documentID).
			Updates(map[string]any{
				"content_markdown": newContent,
				"plain_text":       plainText,
				"updated_by":       userID,
				"updated_at":       now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			contentRecord := &models.DocumentContent{
				ID:              uuid.New(),
				DocumentID:      documentID,
				ContentMarkdown: newContent,
				PlainText:       plainText,
				UpdatedBy:       userID,
			}
			if err := tx.Create(contentRecord).Error; err != nil {
				return err
			}
		}

		updatedAt = now
		return tx.Model(&document).Updates(map[string]any{
			"excerpt":    excerpt,
			"updated_at": now,
			"updated_by": userID,
		}).Error
	})
	if err != nil {
		return nil, err
	}

	return &UpdateContentResult{
		Success:   true,
		UpdatedAt: updatedAt,
	}, nil
}

// CreateInitialContent creates the first content row for a document.
func CreateInitialContent(tx *gorm.DB, documentID, userID uuid.UUID, content string) error {
	contentRecord := &models.DocumentContent{
		ID:              uuid.New(),
		DocumentID:      documentID,
		ContentMarkdown: content,
		PlainText:       toPlainText(content),
		UpdatedBy:       userID,
	}

	return tx.Create(contentRecord).Error
}

// DeleteContentByDocumentID soft deletes the content row for a document.
func DeleteContentByDocumentID(tx *gorm.DB, userID, documentID uuid.UUID) error {
	var count int64
	if err := tx.Model(&models.Document{}).Where("id = ? AND owner_user_id = ?", documentID, userID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return nil
	}

	return tx.Where("document_id = ?", documentID).Delete(&models.DocumentContent{}).Error
}

// RestoreContentByDocumentID restores the content row for a document.
func RestoreContentByDocumentID(tx *gorm.DB, userID, documentID uuid.UUID) error {
	var count int64
	if err := tx.Unscoped().Model(&models.Document{}).Where("id = ? AND owner_user_id = ?", documentID, userID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return nil
	}

	return tx.Unscoped().
		Model(&models.DocumentContent{}).
		Where("document_id = ?", documentID).
		Update("deleted_at", nil).Error
}

// PermanentDeleteContentByDocumentID permanently deletes the content row for a document.
func PermanentDeleteContentByDocumentID(tx *gorm.DB, userID, documentID uuid.UUID) error {
	var count int64
	if err := tx.Unscoped().Model(&models.Document{}).Where("id = ? AND owner_user_id = ?", documentID, userID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return nil
	}

	return tx.Unscoped().Where("document_id = ?", documentID).Delete(&models.DocumentContent{}).Error
}

func currentContentValue(content models.DocumentContent) string {
	if content.ContentMarkdown != "" {
		return content.ContentMarkdown
	}
	return content.ContentJSON
}

func toPlainText(content string) string {
	plainText := content
	plainText = strings.ReplaceAll(plainText, "# ", "")
	plainText = strings.ReplaceAll(plainText, "## ", "")
	plainText = strings.ReplaceAll(plainText, "### ", "")
	plainText = strings.ReplaceAll(plainText, "**", "")
	plainText = strings.ReplaceAll(plainText, "*", "")
	plainText = strings.ReplaceAll(plainText, "[]()", "")
	return strings.TrimSpace(plainText)
}

func buildExcerpt(plainText string) string {
	if len(plainText) == 0 {
		return ""
	}
	if len(plainText) > 100 {
		return plainText[:100] + "..."
	}
	return plainText
}
