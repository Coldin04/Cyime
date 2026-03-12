package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BeforeCreate will set a UUID rather than relying on the database to generate it.
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}

// User represents the core user model
type User struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key"`
	Email       *string   `gorm:"unique"`
	DisplayName *string
	AvatarURL   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// BeforeCreate will set a UUID rather than relying on the database to generate it.
func (a *AuthProvider) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return
}

// AuthProvider stores configuration for an OIDC or OAuth2 provider
type AuthProvider struct {
	ID                    uuid.UUID `gorm:"type:uuid;primary_key"`
	Name                  string    `gorm:"type:varchar(100);not null;unique"`
	ProtocolType          string    `gorm:"type:varchar(20);not null;default:'oidc'"`
	IssuerURL             *string   `gorm:"type:varchar(255)"`
	AuthURL               *string   `gorm:"type:varchar(255)"` // For OAuth2
	TokenURL              *string   `gorm:"type:varchar(255)"` // For OAuth2
	UserInfoURL           *string   `gorm:"type:varchar(255)"`
	ClientID              string    `gorm:"type:varchar(255);not null"`
	ClientSecretEncrypted string    `gorm:"type:text;not null"`
	IconURL               *string   `gorm:"type:varchar(255)"`
	Scopes                string    `gorm:"type:varchar(255);not null"`
	IsActive              bool      `gorm:"not null;default:true"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// BeforeCreate will set a UUID rather than relying on the database to generate it.
func (uip *UserIdentityProvider) BeforeCreate(tx *gorm.DB) (err error) {
	if uip.ID == uuid.Nil {
		uip.ID = uuid.New()
	}
	return
}

// UserIdentityProvider links a user to an OIDC identity
type UserIdentityProvider struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key"`
	UserID         uuid.UUID `gorm:"not null"`
	User           User      `gorm:"foreignKey:UserID"`
	ProviderName   string    `gorm:"type:varchar(100);not null"`
	ProviderUserID string    `gorm:"type:varchar(255);not null"`
	CreatedAt      time.Time

	// Unique constraints
	// _      struct{} `gorm:"uniqueIndex:idx_user_provider,columns:user_id,provider_name"`
	// _      struct{} `gorm:"uniqueIndex:idx_provider_user,columns:provider_name,provider_user_id"`
}

// BeforeCreate will set a UUID rather than relying on the database to generate it.
func (urt *UserRefreshToken) BeforeCreate(tx *gorm.DB) (err error) {
	if urt.ID == uuid.Nil {
		urt.ID = uuid.New()
	}
	return
}

// UserRefreshToken stores a user's long-lived refresh token.
type UserRefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key"`
	UserID    uuid.UUID `gorm:"not null"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	TokenHash string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

// BeforeCreate will set a UUID rather than relying on the database to generate it.
func (f *Folder) BeforeCreate(tx *gorm.DB) (err error) {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return
}

// Folder represents a folder in the workspace
type Folder struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key"`
	OwnerUserID uuid.UUID  `gorm:"not null;index:idx_owner_parent"`
	ParentID    *uuid.UUID `gorm:"index:idx_owner_parent"`
	Name        string     `gorm:"type:varchar(255);not null"`
	Description *string    `gorm:"type:text"`
	CreatedBy   uuid.UUID  `gorm:"not null"`
	UpdatedBy   uuid.UUID  `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate will set a UUID rather than relying on the database to generate it.
func (d *Document) BeforeCreate(tx *gorm.DB) (err error) {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return
}

// Document represents an editable workspace document (metadata only).
type Document struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key"`
	OwnerUserID  uuid.UUID  `gorm:"not null;index:idx_owner_folder"`
	FolderID     *uuid.UUID `gorm:"index:idx_owner_folder"`
	Title        string     `gorm:"type:varchar(255);not null"`
	Excerpt      string     `gorm:"type:text"`
	DocumentType string     `gorm:"type:varchar(50);not null;default:'rich_text'"`
	EditorType   string     `gorm:"type:varchar(50);not null;default:'tiptap'"`
	CreatedBy    uuid.UUID  `gorm:"not null"`
	UpdatedBy    uuid.UUID  `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate will set a UUID rather than relying on the database to generate it.
func (dbd *DocumentBody) BeforeCreate(tx *gorm.DB) (err error) {
	if dbd.ID == uuid.Nil {
		dbd.ID = uuid.New()
	}
	return
}

// DocumentBody stores current canonical editor content for a document.
type DocumentBody struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key"`
	DocumentID     uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex"`
	ContentJSON    string         `gorm:"type:text;not null"`
	PlainText      string         `gorm:"type:text;not null;default:''"`
	ContentVersion int64          `gorm:"not null;default:1"`
	YjsState       string         `gorm:"type:text"`
	YjsStateVector string         `gorm:"type:text"`
	UpdatedBy      uuid.UUID      `gorm:"not null"`
	CreatedAt      time.Time      `gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate will set a UUID rather than relying on the database to generate it.
func (a *Asset) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return
}

// Asset represents a stored binary resource such as an image, video, or file.
type Asset struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key"`
	OwnerUserID     uuid.UUID  `gorm:"not null;index:idx_owner_document"`
	DocumentID      *uuid.UUID `gorm:"type:uuid;index:idx_owner_document"`
	Kind            string     `gorm:"type:varchar(20);not null;default:'image'"`
	Filename        string     `gorm:"type:varchar(255);not null"`
	FileHash        string     `gorm:"type:varchar(64);not null;index"`
	FileSize        int64      `gorm:"not null"`
	MimeType        string     `gorm:"type:varchar(100);not null"`
	StorageProvider string     `gorm:"type:varchar(50);not null;default:'r2'"`
	Bucket          string     `gorm:"type:varchar(255)"`
	ObjectKey       string     `gorm:"type:varchar(255);not null;unique"`
	URL             string     `gorm:"type:text;not null"`
	AltText         *string    `gorm:"type:text"`
	Width           *int       `gorm:"type:int"`
	Height          *int       `gorm:"type:int"`
	Status          string     `gorm:"type:varchar(20);not null;default:'ready'"`
	ReferenceCount  int        `gorm:"not null;default:1;index"`
	CreatedBy       uuid.UUID  `gorm:"not null"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

// Set GORM table names to use snake_case
func (User) TableName() string {
	return "users"
}

func (AuthProvider) TableName() string {
	return "auth_providers"
}

func (UserIdentityProvider) TableName() string {
	return "user_identity_providers"
}

func (UserRefreshToken) TableName() string {
	return "user_refresh_tokens"
}

func (Folder) TableName() string {
	return "folders"
}

func (Document) TableName() string {
	return "documents"
}

func (DocumentBody) TableName() string {
	return "document_bodies"
}

func (Asset) TableName() string {
	return "assets"
}
