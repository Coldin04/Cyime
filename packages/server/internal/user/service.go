package user

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"regexp"
	"strings"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/media"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
)

var githubUsernamePattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37})$`)

func GetUserByID(userID uuid.UUID) (*models.User, error) {
	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
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
