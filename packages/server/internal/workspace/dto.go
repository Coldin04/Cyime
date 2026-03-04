package workspace

import (
	"time"

	"github.com/google/uuid"
)

// CreatorInfo represents the creator information in responses
type CreatorInfo struct {
	ID          uuid.UUID `json:"id"`
	DisplayName *string   `json:"displayName"`
}

// FileItem represents a unified file item (folder or markdown) in the response
type FileItem struct {
	ID          uuid.UUID   `json:"id"`
	Type        string      `json:"type"` // "folder" | "markdown"
	Name        string      `json:"name"`
	Description *string     `json:"description,omitempty"`
	ParentID    *uuid.UUID  `json:"parentId,omitempty"`
	FolderID    *uuid.UUID  `json:"folderId,omitempty"`
	Title       *string     `json:"title,omitempty"`
	Excerpt     *string     `json:"excerpt,omitempty"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
	Creator     CreatorInfo `json:"creator"`
}

// FileListResponse represents the response for the file list API
type FileListResponse struct {
	Items   []FileItem `json:"items"`
	HasMore bool       `json:"hasMore"`
	Total   int64      `json:"total"`
}

// CreateFolderRequest represents the request body for creating a folder
type CreateFolderRequest struct {
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	ParentID    *uuid.UUID `json:"parentId"`
}

// CreateFolderResponse represents the response for creating a folder
type CreateFolderResponse struct {
	ID          uuid.UUID   `json:"id"`
	Type        string      `json:"type"`
	Name        string      `json:"name"`
	Description *string     `json:"description,omitempty"`
	ParentID    *uuid.UUID  `json:"parentId,omitempty"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
	Creator     CreatorInfo `json:"creator"`
}

// CreateMarkdownRequest represents the request body for creating a markdown document
type CreateMarkdownRequest struct {
	Title    string     `json:"title"`
	Content  string     `json:"content"`
	FolderID *uuid.UUID `json:"folderId"`
}

// CreateMarkdownResponse represents the response for creating a markdown document
type CreateMarkdownResponse struct {
	ID        uuid.UUID   `json:"id"`
	Type      string      `json:"type"`
	Title     string      `json:"title"`
	Excerpt   string      `json:"excerpt"`
	FolderID  *uuid.UUID  `json:"folderId,omitempty"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
	Creator   CreatorInfo `json:"creator"`
}

// DeleteResponse represents the response for delete operations
type DeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// AncestorItem represents a single folder in a breadcrumb path
type AncestorItem struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// MoveMarkdownRequest represents the request body for moving a markdown document
type MoveMarkdownRequest struct {
	FolderID *uuid.UUID `json:"folderId"` // null means move to root
}

// MoveFolderRequest represents the request body for moving a folder
type MoveFolderRequest struct {
	ParentID *uuid.UUID `json:"parentId"` // null means move to root
}

// MoveResponse represents the response for move operations
type MoveResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	UpdatedAt time.Time `json:"updatedAt"`
}
