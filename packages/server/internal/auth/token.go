package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JWTClaims represents the claims for our access token
type JWTClaims struct {
	jwt.RegisteredClaims
	UserID uuid.UUID `json:"userId"`
}

type TokenService struct {
	jwtSecret              []byte
	accessTokenLifetime    time.Duration
	refreshTokenLifetime   time.Duration
	refreshTokenByteLength int
}

// NewTokenService creates a new token service, reading configuration from environment variables.
func NewTokenService() (*TokenService, error) {
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		// For development, we can use a default insecure key.
		// In production, this should cause a fatal error.
		// For this project, we'll allow a default for ease of setup.
		secret = "insecure-default-secret-for-dev-only"
		fmt.Println("WARNING: JWT_SECRET_KEY not set, using insecure default key. DO NOT use in production.")
	}

	accessTokenLifetimeMinutes, err := strconv.Atoi(os.Getenv("ACCESS_TOKEN_LIFETIME_MINUTES"))
	if err != nil || accessTokenLifetimeMinutes <= 0 {
		accessTokenLifetimeMinutes = 15 // Default to 15 minutes
	}

	refreshTokenLifetimeHours, err := strconv.Atoi(os.Getenv("REFRESH_TOKEN_LIFETIME_HOURS"))
	if err != nil || refreshTokenLifetimeHours <= 0 {
		refreshTokenLifetimeHours = 720 // Default to 30 days (30 * 24)
	}

	return &TokenService{
		jwtSecret:              []byte(secret),
		accessTokenLifetime:    time.Duration(accessTokenLifetimeMinutes) * time.Minute,
		refreshTokenLifetime:   time.Duration(refreshTokenLifetimeHours) * time.Hour,
		refreshTokenByteLength: 32,
	}, nil
}

// GenerateAndPersistTokens creates new access and refresh tokens, and persists the refresh token hash.
// This must be run within a GORM transaction.
func (s *TokenService) GenerateAndPersistTokens(tx *gorm.DB, user *models.User) (accessTokenString, rawRefreshTokenString string, err error) {
	// 1. Generate Access Token
	accessTokenString, err = s.generateAccessToken(user.ID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// 2. Generate Refresh Token
	rawRefreshTokenString, err = s.generateRefreshToken()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// 3. Hash and Persist Refresh Token
	refreshTokenHash := sha256.Sum256([]byte(rawRefreshTokenString))
	refreshTokenExpiresAt := time.Now().Add(s.refreshTokenLifetime)

	refreshToken := models.UserRefreshToken{
		UserID:    user.ID,
		TokenHash: hex.EncodeToString(refreshTokenHash[:]),
		ExpiresAt: refreshTokenExpiresAt,
	}

	if err := tx.Create(&refreshToken).Error; err != nil {
		return "", "", fmt.Errorf("failed to persist refresh token: %w", err)
	}

	return accessTokenString, rawRefreshTokenString, nil
}

// DeliverTokensAndRedirect sets the refresh token in a secure cookie and redirects the user.
func (s *TokenService) DeliverTokensAndRedirect(c *fiber.Ctx, accessToken, refreshToken string) error {
	// Set the refresh token cookie
	c.Cookie(&fiber.Cookie{
		Name:     "cyime_refresh_token",
		Value:    refreshToken,
		Expires:  time.Now().Add(s.refreshTokenLifetime),
		HTTPOnly: true,
		Secure:   c.Protocol() == "https", // Only set secure flag if on HTTPS
		SameSite: "Lax",
		Path:     "/api/v1/auth", // Important: Scope cookie to the auth path to prevent it being sent on every request
	})

	// 通过将过期时间设置为过去来清除 oidc_state cookie。
	c.Cookie(&fiber.Cookie{
		Name:     "oidc_state",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour), // 设置为一小时前，使其立即过期
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Lax",
		Path:     "/",
	})

	frontendCallbackURL := os.Getenv("FRONTEND_CALLBACK_URL")
	if frontendCallbackURL == "" {
		frontendCallbackURL = "http://localhost:5173/auth/callback" // Default for local dev
	}

	redirectURL := fmt.Sprintf("%s#token=%s", frontendCallbackURL, accessToken)
	return c.Redirect(redirectURL, fiber.StatusTemporaryRedirect)
}

func (s *TokenService) generateAccessToken(userID uuid.UUID) (string, error) {
	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTokenLifetime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "CyimeWrite",
			Subject:   userID.String(),
		},
		UserID: userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *TokenService) generateRefreshToken() (string, error) {
	b := make([]byte, s.refreshTokenByteLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// RevokeRefreshToken deletes a refresh token from the database.
func (s *TokenService) RevokeRefreshToken(rawRefreshToken string) error {
	// Hash the incoming token to look it up in the database.
	tokenHash := sha256.Sum256([]byte(rawRefreshToken))
	tokenHashStr := hex.EncodeToString(tokenHash[:])

	// Delete the token from the database.
	// We use Unscoped() to ensure a hard delete, not a soft delete.
	result := database.DB.Unscoped().Where("token_hash = ?", tokenHashStr).Delete(&models.UserRefreshToken{})

	if result.Error != nil {
		return result.Error
	}

	return nil
}

// HandleRefresh processes a refresh token request, implementing token rotation for security.
func (s *TokenService) HandleRefresh(c *fiber.Ctx) error {
	// 1. Get the refresh token from the secure cookie.
	rawRefreshToken := c.Cookies("cyime_refresh_token")
	if rawRefreshToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Refresh token not found"})
	}

	// 2. Hash the incoming token to look it up in the database.
	incomingTokenHash := sha256.Sum256([]byte(rawRefreshToken))
	incomingTokenHashStr := hex.EncodeToString(incomingTokenHash[:])

	var foundToken models.UserRefreshToken
	var newAccessToken string
	var newRefreshToken string

	// 3. Start a transaction for the rotation logic.
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Find the token and ensure it's not expired.
		if err := tx.Preload("User").Where("token_hash = ? AND expires_at > ?", incomingTokenHashStr, time.Now()).First(&foundToken).Error; err != nil {
			// If not found (or another DB error), the token is invalid.
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid or expired refresh token")
		}

		// --- TOKEN ROTATION ---
		// Immediately delete the used token.
		if err := tx.Delete(&foundToken).Error; err != nil {
			return fmt.Errorf("failed to delete used refresh token: %w", err)
		}

		// Generate a new access token.
		var err error
		newAccessToken, err = s.generateAccessToken(foundToken.UserID)
		if err != nil {
			return fmt.Errorf("failed to generate new access token: %w", err)
		}
		
		// Generate and persist a new refresh token.
		newRefreshToken, err = s.generateRefreshToken()
		if err != nil {
			return fmt.Errorf("failed to generate new refresh token: %w", err)
		}

		newRefreshTokenHash := sha256.Sum256([]byte(newRefreshToken))
		newRefreshTokenExpiresAt := time.Now().Add(s.refreshTokenLifetime)

		replacementToken := models.UserRefreshToken{
			UserID:    foundToken.UserID,
			TokenHash: hex.EncodeToString(newRefreshTokenHash[:]),
			ExpiresAt: newRefreshTokenExpiresAt,
		}
		if err := tx.Create(&replacementToken).Error; err != nil {
			return fmt.Errorf("failed to persist new refresh token: %w", err)
		}

		return nil // Commit transaction
	})

	if err != nil {
		// If the transaction failed, it's either an internal error or the token was invalid.
		// In either case, clear the potentially invalid cookie on the client.
		c.ClearCookie("cyime_refresh_token")
		// Use the error from fiber.NewError if it exists
		if e, ok := err.(*fiber.Error); ok {
			return c.Status(e.Code).JSON(fiber.Map{"error": e.Message})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// 4. Send the new refresh token in a new secure cookie.
	c.Cookie(&fiber.Cookie{
		Name:     "cyime_refresh_token",
		Value:    newRefreshToken,
		Expires:  time.Now().Add(s.refreshTokenLifetime),
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Lax",
		Path:     "/api/v1/auth",
	})

	// 5. Send the new access token in the response body.
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"accessToken": newAccessToken,
	})
}
