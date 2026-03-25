package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/url"
	"regexp"
	"strings"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/config"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/media"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var githubUsernamePattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37})$`)

// OverviewStats stores the lightweight numbers shown in the user overview panel.
type OverviewStats struct {
	ActiveDocumentCount  int64
	TrashedDocumentCount int64
	DocumentLimit        *int
	Unlimited            bool
}

const (
	ImageBedProviderSEE  = "see"
	ImageBedProviderLsky = "lsky"
)

func GetUserByID(userID uuid.UUID) (*models.User, error) {
	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetEffectiveDocumentQuota resolves the effective document limit for one user.
// 优先使用用户自己的配额；如果用户没有单独配置，则回退到全局默认值；都没有时表示无限制。
func GetEffectiveDocumentQuota(userID uuid.UUID) (*int, error) {
	currentUser, err := GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	if currentUser.DocumentQuota != nil {
		return currentUser.DocumentQuota, nil
	}

	return config.GetOptionalNonNegativeInt("DEFAULT_DOCUMENT_QUOTA")
}

// GetOverviewStats returns overview document counts for the current user.
func GetOverviewStats(userID uuid.UUID) (*OverviewStats, error) {
	limit, err := GetEffectiveDocumentQuota(userID)
	if err != nil {
		return nil, err
	}

	var activeCount int64
	if err := database.DB.Model(&models.Document{}).
		Where("owner_user_id = ? AND deleted_at IS NULL", userID).
		Count(&activeCount).Error; err != nil {
		return nil, err
	}

	var trashedCount int64
	if err := database.DB.Unscoped().Model(&models.Document{}).
		Where("owner_user_id = ? AND deleted_at IS NOT NULL", userID).
		Count(&trashedCount).Error; err != nil {
		return nil, err
	}

	return &OverviewStats{
		ActiveDocumentCount:  activeCount,
		TrashedDocumentCount: trashedCount,
		DocumentLimit:        limit,
		Unlimited:            limit == nil,
	}, nil
}

func UpdateProfile(userID uuid.UUID, displayName string) (*models.User, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, errors.New("displayName is required")
	}
	if len([]rune(displayName)) > 80 {
		return nil, errors.New("displayName is too long")
	}

	if err := database.DB.Model(&models.User{}).
		Where("id = ?", userID).
		Update("display_name", displayName).Error; err != nil {
		return nil, err
	}

	return GetUserByID(userID)
}

type ImageBedConfig struct {
	ID           uuid.UUID
	Name         string
	ProviderType string
	BaseURL      string
	APIToken     string
	IsEnabled    bool
	StorageID    int
	StrategyID   string
}

type UpsertImageBedConfigInput struct {
	Name         string
	ProviderType string
	BaseURL      string
	APIToken     string
	IsEnabled    bool
	StorageID    int
	StrategyID   string
}

type imageBedConfigExtras struct {
	StorageID  int    `json:"storageId,omitempty"`
	StrategyID string `json:"strategyId,omitempty"`
}

func ListImageBedConfigs(userID uuid.UUID) ([]ImageBedConfig, error) {
	var rows []models.UserImageBedConfig
	if err := database.DB.
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]ImageBedConfig, 0, len(rows))
	for _, row := range rows {
		items = append(items, imageBedModelToConfig(row))
	}
	return items, nil
}

func CreateImageBedConfig(userID uuid.UUID, input UpsertImageBedConfigInput) (*ImageBedConfig, error) {
	normalized, err := normalizeImageBedConfigInput(input)
	if err != nil {
		return nil, err
	}

	row := models.UserImageBedConfig{
		UserID:       userID,
		Name:         normalized.Name,
		ProviderType: normalized.ProviderType,
		BaseURL:      stringPtrOrNil(normalized.BaseURL),
		APIToken:     stringPtrOrNil(normalized.APIToken),
		ConfigJSON:   stringPtrOrNil(buildImageBedConfigJSON(normalized.StorageID, normalized.StrategyID)),
		IsEnabled:    normalized.IsEnabled,
	}
	if err := database.DB.Create(&row).Error; err != nil {
		return nil, err
	}

	config := imageBedModelToConfig(row)
	return &config, nil
}

func UpdateImageBedConfig(userID uuid.UUID, configID uuid.UUID, input UpsertImageBedConfigInput) (*ImageBedConfig, error) {
	normalized, err := normalizeImageBedConfigInput(input)
	if err != nil {
		return nil, err
	}

	var row models.UserImageBedConfig
	if err := database.DB.
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", configID, userID).
		First(&row).Error; err != nil {
		return nil, err
	}

	updates := map[string]any{
		"name":          normalized.Name,
		"provider_type": normalized.ProviderType,
		"base_url":      nullableTrimmedString(normalized.BaseURL),
		"api_token":     nullableTrimmedString(normalized.APIToken),
		"config_json":   nullableTrimmedString(buildImageBedConfigJSON(normalized.StorageID, normalized.StrategyID)),
		"is_enabled":    normalized.IsEnabled,
	}
	if err := database.DB.Model(&row).Updates(updates).Error; err != nil {
		return nil, err
	}

	if err := database.DB.
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", configID, userID).
		First(&row).Error; err != nil {
		return nil, err
	}

	config := imageBedModelToConfig(row)
	return &config, nil
}

func DeleteImageBedConfig(userID uuid.UUID, configID uuid.UUID) error {
	result := database.DB.
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", configID, userID).
		Delete(&models.UserImageBedConfig{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func GetImageBedConfigByID(userID uuid.UUID, configID uuid.UUID) (*ImageBedConfig, error) {
	var row models.UserImageBedConfig
	if err := database.DB.
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", configID, userID).
		First(&row).Error; err != nil {
		return nil, err
	}

	config := imageBedModelToConfig(row)
	return &config, nil
}

func normalizeImageBedConfigInput(input UpsertImageBedConfigInput) (*UpsertImageBedConfigInput, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("image bed name is required")
	}
	if len([]rune(name)) > 120 {
		return nil, errors.New("image bed name is too long")
	}

	providerType := strings.TrimSpace(input.ProviderType)
	switch providerType {
	case ImageBedProviderSEE:
		if strings.TrimSpace(input.APIToken) == "" {
			return nil, errors.New("S.EE API token is required")
		}
	case ImageBedProviderLsky:
		baseURL := strings.TrimSpace(input.BaseURL)
		if baseURL == "" {
			return nil, errors.New("Lsky API url is required")
		}
		parsed, err := url.Parse(baseURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return nil, errors.New("invalid lsky api url")
		}
		input.BaseURL = strings.TrimRight(parsed.String(), "/")
		if strings.TrimSpace(input.APIToken) == "" {
			return nil, errors.New("Lsky API token is required")
		}
		if input.StorageID <= 0 {
			return nil, errors.New("Lsky storage id is required")
		}
	default:
		return nil, errors.New("unsupported image bed provider")
	}

	if providerType != ImageBedProviderLsky {
		input.BaseURL = ""
		input.StorageID = 0
		input.StrategyID = ""
	}

	input.Name = name
	input.ProviderType = providerType
	input.APIToken = strings.TrimSpace(input.APIToken)
	input.StrategyID = strings.TrimSpace(input.StrategyID)
	return &input, nil
}

func imageBedModelToConfig(row models.UserImageBedConfig) ImageBedConfig {
	extras := parseImageBedConfigJSON(row.ConfigJSON)
	return ImageBedConfig{
		ID:           row.ID,
		Name:         row.Name,
		ProviderType: row.ProviderType,
		BaseURL:      trimStringPtr(row.BaseURL),
		APIToken:     trimStringPtr(row.APIToken),
		IsEnabled:    row.IsEnabled,
		StorageID:    extras.StorageID,
		StrategyID:   extras.StrategyID,
	}
}

func parseImageBedConfigJSON(value *string) imageBedConfigExtras {
	if value == nil || strings.TrimSpace(*value) == "" {
		return imageBedConfigExtras{}
	}

	var extras imageBedConfigExtras
	if err := json.Unmarshal([]byte(*value), &extras); err != nil {
		return imageBedConfigExtras{}
	}
	return extras
}

func buildImageBedConfigJSON(storageID int, strategyID string) string {
	extras := imageBedConfigExtras{
		StorageID:  storageID,
		StrategyID: strings.TrimSpace(strategyID),
	}
	if extras.StrategyID == "" && extras.StorageID == 0 {
		return ""
	}

	payload, err := json.Marshal(extras)
	if err != nil {
		return ""
	}
	return string(payload)
}

func ResolveAvatarURL(baseURL string, user *models.User) (*string, error) {
	avatarURL := trimStringPtr(user.AvatarURL)
	avatarObjectKey := trimStringPtr(user.AvatarObjectKey)
	if avatarURL == "" {
		return nil, nil
	}
	if avatarObjectKey == "" {
		return &avatarURL, nil
	}

	tokenService, err := media.NewTokenService()
	if err != nil {
		// Degrade gracefully: keep existing avatar URL instead of breaking /user/me.
		return &avatarURL, nil
	}
	token, _, err := tokenService.IssueAvatarReadToken(user.ID, avatarObjectKey)
	if err != nil {
		return &avatarURL, nil
	}

	resolved := strings.TrimRight(baseURL, "/") + "/api/v1/user/avatar/content?token=" + token
	return &resolved, nil
}

func UpdateAvatarWithUpload(ctx context.Context, userID uuid.UUID, fileHeader *multipart.FileHeader) (*models.User, error) {
	currentUser, err := GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	uploadResult, err := media.UploadUserAvatar(ctx, userID, fileHeader)
	if err != nil {
		return nil, err
	}

	oldObjectKey := trimStringPtr(currentUser.AvatarObjectKey)
	newURL := uploadResult.URL
	newObjectKey := uploadResult.ObjectKey
	if err := database.DB.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"avatar_url":        newURL,
			"avatar_object_key": newObjectKey,
		}).Error; err != nil {
		_ = media.DeleteStoredObject(ctx, newObjectKey)
		return nil, err
	}

	if oldObjectKey != "" && oldObjectKey != newObjectKey {
		if err := media.DeleteStoredObject(ctx, oldObjectKey); err != nil {
			log.Printf("[user.avatar] cleanup old uploaded avatar failed user=%s objectKey=%q err=%v", userID, oldObjectKey, err)
		}
	}

	return GetUserByID(userID)
}

func UpdateAvatarWithGitHub(ctx context.Context, userID uuid.UUID, username string) (*models.User, error) {
	currentUser, err := GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	username = strings.TrimSpace(username)
	if username == "" {
		return nil, errors.New("github username is required")
	}
	if !githubUsernamePattern.MatchString(username) {
		return nil, errors.New("invalid github username")
	}

	avatarURL := fmt.Sprintf("https://github.com/%s.png", username)
	oldObjectKey := trimStringPtr(currentUser.AvatarObjectKey)
	if err := database.DB.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"avatar_url":        avatarURL,
			"avatar_object_key": nil,
		}).Error; err != nil {
		return nil, err
	}

	if oldObjectKey != "" {
		if err := media.DeleteStoredObject(ctx, oldObjectKey); err != nil {
			log.Printf("[user.avatar] cleanup replaced uploaded avatar failed user=%s objectKey=%q err=%v", userID, oldObjectKey, err)
		}
	}

	return GetUserByID(userID)
}

func trimStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func nullableTrimmedString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func stringPtrOrNil(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
