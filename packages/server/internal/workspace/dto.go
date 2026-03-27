package workspace

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CreatorInfo represents the creator information in responses.
type CreatorInfo struct {
	ID          uuid.UUID `json:"id"`
	DisplayName *string   `json:"displayName"`
}

// FileItem represents a unified file item (folder or document) in the response.
type FileItem struct {
	ID                     uuid.UUID   `json:"id"`
	Type                   string      `json:"type"` // "folder" | "document"
	DocumentType           *string     `json:"documentType,omitempty"`
	PreferredImageTargetID *string     `json:"preferredImageTargetId,omitempty"`
	Name                   string      `json:"name"`
	Description            *string     `json:"description,omitempty"`
	ParentID               *uuid.UUID  `json:"parentId,omitempty"`
	FolderID               *uuid.UUID  `json:"folderId,omitempty"`
	Title                  *string     `json:"title,omitempty"`
	Excerpt                *string     `json:"excerpt,omitempty"`
	CreatedAt              time.Time   `json:"createdAt"`
	UpdatedAt              time.Time   `json:"updatedAt"`
	Creator                CreatorInfo `json:"creator"`
}

type FileListResponse struct {
	Items   []FileItem `json:"items"`
	HasMore bool       `json:"hasMore"`
	Total   int64      `json:"total"`
}

type CreateFolderRequest struct {
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	ParentID    *uuid.UUID `json:"parentId"`
}

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

type CreateDocumentRequest struct {
	Title                  string          `json:"title"`
	ContentJSON            json.RawMessage `json:"contentJson"`
	FolderID               *uuid.UUID      `json:"folderId"`
	DocumentType           string          `json:"documentType"`
	PreferredImageTargetID string          `json:"preferredImageTargetId"`
}

type CreateDocumentResponse struct {
	ID                     uuid.UUID   `json:"id"`
	Type                   string      `json:"type"`
	DocumentType           string      `json:"documentType"`
	PreferredImageTargetID string      `json:"preferredImageTargetId"`
	Title                  string      `json:"title"`
	Excerpt                string      `json:"excerpt"`
	FolderID               *uuid.UUID  `json:"folderId,omitempty"`
	CreatedAt              time.Time   `json:"createdAt"`
	UpdatedAt              time.Time   `json:"updatedAt"`
	Creator                CreatorInfo `json:"creator"`
}

type DeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type BatchDeleteRequest struct {
	Items []ItemToDelete `json:"items"`
}

type ItemToDelete struct {
	ID   uuid.UUID `json:"id"`
	Type string    `json:"type"` // "folder" | "document"
}

type BatchDeleteResponse struct {
	Success     bool         `json:"success"`
	Message     string       `json:"message"`
	FailedItems []FailedItem `json:"failedItems,omitempty"`
}

type FailedItem struct {
	ID     uuid.UUID `json:"id"`
	Type   string    `json:"type"`
	Reason string    `json:"reason"`
}

type AncestorItem struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type MoveDocumentRequest struct {
	FolderID *uuid.UUID `json:"folderId"`
}

type MoveFolderRequest struct {
	ParentID *uuid.UUID `json:"parentId"`
}

type MoveResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type UpdateDocumentImageTargetRequest struct {
	PreferredImageTargetID string `json:"preferredImageTargetId"`
}

type BatchMoveRequest struct {
	Items               []ItemToMove `json:"items"`
	DestinationFolderID *uuid.UUID   `json:"destinationFolderId"`
}

type ItemToMove struct {
	ID   uuid.UUID `json:"id"`
	Type string    `json:"type"` // "folder" or "document"
}

type BatchMoveResponse struct {
	Success     bool         `json:"success"`
	Message     string       `json:"message"`
	MovedCount  int          `json:"movedCount"`
	FailedItems []FailedItem `json:"failedItems,omitempty"`
}

type ShareDocumentRequest struct {
	UserID uuid.UUID `json:"userId"`
	Role   string    `json:"role"`
}

type ShareDocumentMember struct {
	UserID      uuid.UUID `json:"userId"`
	Role        string    `json:"role"`
	DisplayName *string   `json:"displayName,omitempty"`
}

type ShareDocumentResponse struct {
	DocumentID uuid.UUID             `json:"documentId"`
	Members    []ShareDocumentMember `json:"members"`
}

type SharedDocumentItem struct {
	DocumentID             uuid.UUID  `json:"documentId"`
	Title                  string     `json:"title"`
	Excerpt                string     `json:"excerpt"`
	DocumentType           string     `json:"documentType"`
	PreferredImageTargetID string     `json:"preferredImageTargetId"`
	FolderID               *uuid.UUID `json:"folderId,omitempty"`
	OwnerUserID            uuid.UUID  `json:"ownerUserId"`
	OwnerDisplayName       *string    `json:"ownerDisplayName,omitempty"`
	MyRole                 string     `json:"myRole"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
}

type SharedDocumentListResponse struct {
	Items   []SharedDocumentItem `json:"items"`
	HasMore bool                 `json:"hasMore"`
	Total   int64                `json:"total"`
}
