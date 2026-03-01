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
	ID                      uuid.UUID `gorm:"type:uuid;primary_key"`
	Name                    string    `gorm:"type:varchar(100);not null;unique"`
	ProtocolType            string    `gorm:"type:varchar(20);not null;default:'oidc'"`
	IssuerURL               *string   `gorm:"type:varchar(255)"`
	AuthURL                 *string   `gorm:"type:varchar(255)"` // For OAuth2
	TokenURL                *string   `gorm:"type:varchar(255)"` // For OAuth2
	UserInfoURL             *string   `gorm:"type:varchar(255)"`
	ClientID                string    `gorm:"type:varchar(255);not null"`
	ClientSecretEncrypted   string    `gorm:"type:text;not null"`
	IconURL                 *string   `gorm:"type:varchar(255)"`
	Scopes                  string    `gorm:"type:varchar(255);not null"`
	IsActive                bool      `gorm:"not null;default:true"`
	CreatedAt               time.Time
	UpdatedAt               time.Time
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
	ID               uuid.UUID `gorm:"type:uuid;primary_key"`
	UserID           uuid.UUID `gorm:"not null"`
	User             User      `gorm:"foreignKey:UserID"`
	ProviderName     string    `gorm:"type:varchar(100);not null"`
	ProviderUserID   string    `gorm:"type:varchar(255);not null"`
	CreatedAt        time.Time

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
	ID          uuid.UUID      `gorm:"type:uuid;primary_key"`
	UserID      uuid.UUID      `gorm:"not null;index:idx_user_parent"`
	ParentID    *uuid.UUID     `gorm:"index:idx_user_parent"`
	Name        string         `gorm:"type:varchar(255);not null"`
	Description *string        `gorm:"type:text"`
	CreatedBy   uuid.UUID      `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate will set a UUID rather than relying on the database to generate it.
func (m *Markdown) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return
}

// Markdown represents a markdown document in the workspace
type Markdown struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key"`
	UserID    uuid.UUID      `gorm:"not null;index:idx_user_folder"`
	FolderID  *uuid.UUID     `gorm:"index:idx_user_folder"`
	Title     string         `gorm:"type:varchar(255);not null"`
	Excerpt   string         `gorm:"type:text"`
	Content   string         `gorm:"type:longtext"`
	CreatedBy uuid.UUID      `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
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

func (Markdown) TableName() string {
	return "markdowns"
}
