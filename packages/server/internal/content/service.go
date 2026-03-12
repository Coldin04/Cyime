package content

import (
	"encoding/json"
	"errors"
	"time"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultContentJSON = `{"type":"doc","content":[{"type":"paragraph"}]}`

// GetContentResult represents the current content of a document.
type GetContentResult struct {
	ID             uuid.UUID       `json:"id"`
	DocumentID     uuid.UUID       `json:"documentId"`
	ContentJSON    json.RawMessage `json:"contentJson"`
	PlainText      string          `json:"plainText"`
	ContentVersion int64           `json:"contentVersion"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

// UpdateContentResult represents the result of updating document content.
type UpdateContentResult struct {
	Success        bool      `json:"success"`
	ContentVersion int64     `json:"contentVersion"`
	UpdatedAt      time.Time `json:"updatedAt"`
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

	var content models.DocumentBody
	result = database.DB.Where("document_id = ?", documentID).First(&content)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("文档内容不存在")
		}
		return nil, result.Error
	}

	return &GetContentResult{
		ID:             content.ID,
		DocumentID:     content.DocumentID,
		ContentJSON:    json.RawMessage(content.ContentJSON),
		PlainText:      content.PlainText,
		ContentVersion: content.ContentVersion,
		CreatedAt:      content.CreatedAt,
		UpdatedAt:      content.UpdatedAt,
	}, nil
}

// UpdateContent updates the current content of a document in place.
func UpdateContent(userID uuid.UUID, documentID uuid.UUID, contentJSONRaw []byte) (*UpdateContentResult, error) {
	contentJSON, err := normalizeContentJSON(contentJSONRaw)
	if err != nil {
		return nil, err
	}

	var (
		updatedAt      time.Time
		contentVersion int64
	)
	plainText := toPlainText(contentJSON)
	excerpt := buildExcerpt(plainText)

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var document models.Document
		result := tx.Where("id = ? AND owner_user_id = ?", documentID, userID).First(&document)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return errors.New("文档不存在或无权访问")
			}
			return result.Error
		}

		now := time.Now()
		result = tx.Model(&models.DocumentBody{}).
			Where("document_id = ?", documentID).
			Updates(map[string]any{
				"content_json":    contentJSON,
				"plain_text":      plainText,
				"updated_by":      userID,
				"content_version": gorm.Expr("content_version + 1"),
				"updated_at":      now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			contentRecord := &models.DocumentBody{
				ID:             uuid.New(),
				DocumentID:     documentID,
				ContentJSON:    contentJSON,
				PlainText:      plainText,
				ContentVersion: 1,
				UpdatedBy:      userID,
			}
			if err := tx.Create(contentRecord).Error; err != nil {
				return err
			}
			contentVersion = contentRecord.ContentVersion
		}

		if contentVersion == 0 {
			var body models.DocumentBody
			if err := tx.Where("document_id = ?", documentID).First(&body).Error; err != nil {
				return err
			}
			contentVersion = body.ContentVersion
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
		Success:        true,
		ContentVersion: contentVersion,
		UpdatedAt:      updatedAt,
	}, nil
}

// CreateInitialContent creates the first content row for a document.
func CreateInitialContent(tx *gorm.DB, documentID, userID uuid.UUID, contentJSONRaw string) error {
	contentJSON, err := normalizeContentJSON([]byte(contentJSONRaw))
	if err != nil {
		return err
	}

	contentRecord := &models.DocumentBody{
		ID:             uuid.New(),
		DocumentID:     documentID,
		ContentJSON:    contentJSON,
		PlainText:      toPlainText(contentJSON),
		ContentVersion: 1,
		UpdatedBy:      userID,
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

	return tx.Where("document_id = ?", documentID).Delete(&models.DocumentBody{}).Error
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
		Model(&models.DocumentBody{}).
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

	return tx.Unscoped().Where("document_id = ?", documentID).Delete(&models.DocumentBody{}).Error
}

func normalizeContentJSON(raw []byte) (string, error) {
	if len(raw) == 0 {
		return defaultContentJSON, nil
	}
	if !json.Valid(raw) {
		return "", errors.New("contentJson must be valid JSON")
	}
	return string(raw), nil
}

func toPlainText(contentJSON string) string {
	var node any
	if err := json.Unmarshal([]byte(contentJSON), &node); err != nil {
		return ""
	}

	parts := make([]string, 0, 64)
	collectText(node, &parts)
	return joinWithSpace(parts)
}

func collectText(node any, out *[]string) {
	switch v := node.(type) {
	case map[string]any:
		if text, ok := v["text"].(string); ok {
			*out = append(*out, text)
		}
		if children, ok := v["content"].([]any); ok {
			for _, child := range children {
				collectText(child, out)
			}
		}
	case []any:
		for _, item := range v {
			collectText(item, out)
		}
	}
}

func joinWithSpace(parts []string) string {
	merged := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		if merged == "" {
			merged = p
			continue
		}
		merged += " " + p
	}
	return merged
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

// BuildExcerptFromContentJSON derives a document excerpt from editor JSON.
func BuildExcerptFromContentJSON(contentJSONRaw string) string {
	contentJSON, err := normalizeContentJSON([]byte(contentJSONRaw))
	if err != nil {
		return ""
	}
	return buildExcerpt(toPlainText(contentJSON))
}
