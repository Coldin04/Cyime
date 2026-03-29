package workspace

import (
	"errors"
	"os"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/acl"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/config"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func ensureSharingEnabledForUser(tx *gorm.DB, userID uuid.UUID) error {
	if config.IsTrue(os.Getenv("SHARING_DEV_BYPASS")) {
		return nil
	}

	if !config.IsTrue(os.Getenv("SMTP_ENABLED")) {
		return errors.New("邮箱邀请功能未启用")
	}

	var user models.User
	if err := tx.Select("id", "email_verified").Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return err
	}
	if !user.EmailVerified {
		return errors.New("邮箱未验证，暂不可使用共享功能")
	}
	return nil
}

func loadShareManagedDocument(tx *gorm.DB, actorUserID, documentID uuid.UUID) (*models.Document, string, error) {
	document, role, err := acl.CanManageDocumentMembers(tx, actorUserID, documentID)
	if err != nil {
		return nil, "", errors.New("文档不存在或无权访问")
	}
	return document, role, nil
}

func getInviteCooldownConfig() (baseSeconds, multiplier, maxSeconds int) {
	baseSeconds = 60
	multiplier = 2
	maxSeconds = 86400

	if parsed, err := config.GetOptionalNonNegativeInt("INVITE_COOLDOWN_BASE_SECONDS"); err == nil && parsed != nil && *parsed > 0 {
		baseSeconds = *parsed
	}
	if parsed, err := config.GetOptionalNonNegativeInt("INVITE_COOLDOWN_MULTIPLIER"); err == nil && parsed != nil && *parsed > 0 {
		multiplier = *parsed
	}
	if parsed, err := config.GetOptionalNonNegativeInt("INVITE_COOLDOWN_MAX_SECONDS"); err == nil && parsed != nil && *parsed > 0 {
		maxSeconds = *parsed
	}
	if maxSeconds < baseSeconds {
		maxSeconds = baseSeconds
	}
	return
}
