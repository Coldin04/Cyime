package ai

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"g.co1d.in/Coldin04/Cyime/server/internal/content"
	"g.co1d.in/Coldin04/Cyime/server/internal/database"
	"g.co1d.in/Coldin04/Cyime/server/internal/editlease"
	"g.co1d.in/Coldin04/Cyime/server/internal/workspace"
	"github.com/google/uuid"
)

var (
	ErrUnsupportedFormat = errors.New("unsupported content format")
	ErrVersionConflict   = errors.New("document changed while applying update")
)

type MarkdownContent struct {
	Format    string    `json:"format"`
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type MarkdownUpdateResult struct {
	Success   bool      `json:"success"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateMarkdownDocumentInput struct {
	Title                  string
	Content                string
	FolderID               *uuid.UUID
	PreferredImageTargetID string
}

type CreateMarkdownDocumentResult struct {
	ID        uuid.UUID  `json:"id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	FolderID  *uuid.UUID `json:"folderId,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type markdownContentSnapshot struct {
	Format    string
	Content   string
	Version   int64
	UpdatedAt time.Time
}

func GetMarkdownContent(userID uuid.UUID, documentID uuid.UUID) (*MarkdownContent, error) {
	snapshot, err := getMarkdownContentSnapshot(userID, documentID)
	if err != nil {
		return nil, err
	}
	return &MarkdownContent{
		Format:    snapshot.Format,
		Content:   snapshot.Content,
		UpdatedAt: snapshot.UpdatedAt,
	}, nil
}

func getMarkdownContentSnapshot(userID uuid.UUID, documentID uuid.UUID) (*markdownContentSnapshot, error) {
	result, err := content.GetContent(userID, documentID)
	if err != nil {
		return nil, err
	}
	markdown, err := contentJSONToMarkdown(result.ContentJSON)
	if err != nil {
		return nil, err
	}
	return &markdownContentSnapshot{
		Format:    "markdown",
		Content:   markdown,
		Version:   result.ContentVersion,
		UpdatedAt: result.UpdatedAt,
	}, nil
}

func UpdateMarkdownContent(userID uuid.UUID, documentID uuid.UUID, markdown string) (*MarkdownUpdateResult, error) {
	return updateMarkdownContent(userID, documentID, markdown, nil)
}

func updateMarkdownContent(userID uuid.UUID, documentID uuid.UUID, markdown string, expectedVersion *int64) (*MarkdownUpdateResult, error) {
	contentJSON, err := markdownToContentJSON(markdown)
	if err != nil {
		return nil, err
	}

	manager := editlease.New(database.DB)
	grant, err := manager.Claim(userID, documentID, "api-operation-"+uuid.NewString(), "", false)
	if err != nil {
		return nil, err
	}
	defer func() { _ = manager.Release(userID, documentID, grant.Token) }()

	version := expectedVersion
	if version == nil {
		current, err := content.GetContent(userID, documentID)
		if err != nil {
			return nil, err
		}
		version = &current.ContentVersion
	}

	result, err := content.UpdateContent(userID, documentID, contentJSON, grant.Token, *version)
	var conflict *content.ContentVersionConflictError
	if errors.As(err, &conflict) {
		return nil, ErrVersionConflict
	}
	if err != nil {
		return nil, err
	}
	return &MarkdownUpdateResult{
		Success:   result.Success,
		UpdatedAt: result.UpdatedAt,
	}, nil
}

func PatchMarkdownContent(userID uuid.UUID, documentID uuid.UUID, operations []PatchOperation) (*MarkdownUpdateResult, error) {
	current, err := getMarkdownContentSnapshot(userID, documentID)
	if err != nil {
		return nil, err
	}

	patched, err := applyMarkdownPatch(current.Content, operations)
	if err != nil {
		return nil, err
	}
	return updateMarkdownContent(userID, documentID, patched, &current.Version)
}

func RenameDocument(userID, documentID uuid.UUID, title string) error {
	manager := editlease.New(database.DB)
	grant, err := manager.Claim(userID, documentID, "api-operation-"+uuid.NewString(), "", false)
	if err != nil {
		return err
	}
	defer func() { _ = manager.Release(userID, documentID, grant.Token) }()
	return workspace.UpdateDocumentTitle(userID, documentID, title)
}

func CreateMarkdownDocument(userID uuid.UUID, input CreateMarkdownDocumentInput) (*CreateMarkdownDocumentResult, error) {
	contentJSON, err := markdownToContentJSON(input.Content)
	if err != nil {
		return nil, err
	}
	document, err := workspace.CreateDocument(
		userID,
		input.Title,
		string(contentJSON),
		input.FolderID,
		"rich_text",
		input.PreferredImageTargetID,
	)
	if err != nil {
		return nil, err
	}

	return &CreateMarkdownDocumentResult{
		ID:        document.ID,
		Type:      "document",
		Title:     document.Title,
		FolderID:  document.FolderID,
		CreatedAt: document.CreatedAt,
		UpdatedAt: document.UpdatedAt,
	}, nil
}

func normalizeFormat(format string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(format))
	if normalized == "" {
		normalized = "markdown"
	}
	if normalized != "markdown" {
		return "", ErrUnsupportedFormat
	}
	return normalized, nil
}

func parseDocumentID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid document id")
	}
	return id, nil
}
