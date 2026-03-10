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
	UserID      uuid.UUID  `gorm:"not null;index:idx_user_parent"`
	ParentID    *uuid.UUID `gorm:"index:idx_user_parent"`
	Name        string     `gorm:"type:varchar(255);not null"`
	Description *string    `gorm:"type:text"`
	CreatedBy   uuid.UUID  `gorm:"not null"`
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

// Markdown represents a markdown document in the workspace (metadata only)
type Markdown struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key"`
	UserID    uuid.UUID  `gorm:"not null;index:idx_user_folder"`
	FolderID  *uuid.UUID `gorm:"index:idx_user_folder"`
	Title     string     `gorm:"type:varchar(255);not null"`
	Excerpt   string     `gorm:"type:text"` // 纯文本摘要（前 100 字）
	CreatedBy uuid.UUID  `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate will set a UUID rather than relying on the database to generate it.
func (mc *MarkdownContent) BeforeCreate(tx *gorm.DB) (err error) {
	if mc.ID == uuid.Nil {
		mc.ID = uuid.New()
	}
	return
}

// MarkdownContent represents the content of a markdown document (supports version history)
type MarkdownContent struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key"`                         // 独立 UUID
	MarkdownID uuid.UUID      `gorm:"type:uuid;not null;index:idx_markdown_version"` // 关联文档 ID
	Version    int            `gorm:"not null;default:1;index:idx_markdown_version"` // 版本号
	Content    string         `gorm:"type:longtext"`                                 // 完整 Markdown 内容
	CreatedAt  time.Time      `gorm:"autoCreateTime"`                                // 版本创建时间
	DeletedAt  gorm.DeletedAt `gorm:"index"`                                         // 软删除，便于回收站恢复正文
}

// BeforeCreate will set a UUID rather than relying on the database to generate it.
func (a *Attachment) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return
}

// Attachment represents an attachment (image, file) uploaded by a user
type Attachment struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key"`
	UserID         uuid.UUID  `gorm:"not null;index:idx_user_markdown"`
	MarkdownID     *uuid.UUID `gorm:"type:uuid;index:idx_user_markdown"` // 所属文档（NULL=未关联）
	Filename       string     `gorm:"type:varchar(255);not null"`        // 原始文件名
	FileHash       string     `gorm:"type:varchar(64);not null;index"`   // SHA256 Hash（去重用）
	FileSize       int64      `gorm:"not null"`                          // 文件大小 (bytes)
	MimeType       string     `gorm:"type:varchar(100);not null"`        // MIME 类型
	R2Key          string     `gorm:"type:varchar(255);not null;unique"` // R2 对象键
	R2URL          string     `gorm:"type:text;not null"`                // R2 公开访问 URL
	ReferenceCount int        `gorm:"not null;default:1;index"`          // 引用计数（去重用）
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"` // 软删除，用于垃圾回收
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

func (MarkdownContent) TableName() string {
	return "markdown_contents"
}

func (Attachment) TableName() string {
	return "attachments"
}
