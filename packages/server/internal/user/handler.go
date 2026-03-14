package user

import (
	"context"
	"errors"
	"io"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/media"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// UserResponseDTO defines the data structure for the user profile response.
// This prevents leaking unwanted or sensitive fields from the database model.
type UserResponseDTO struct {
	ID          uuid.UUID `json:"id"`
	Email       *string   `json:"email"`
	DisplayName *string   `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl"`
}

type UpdateProfileRequest struct {
	DisplayName string `json:"displayName"`
}

type UpdateGitHubAvatarRequest struct {
	Username string `json:"username"`
}

// GetMe handles the GET /api/v1/user/me request.
// It relies on the Protected middleware to have already validated the JWT
// and placed the userId in the context.
func GetMe(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: Invalid token context.",
		})
	}

	user, err := GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found.",
		})
	}

	response, err := toUserResponseDTO(c, user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(response)
}

func UpdateProfileHandler(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID format in token.",
		})
	}

	var req UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body.",
		})
	}

	user, err := UpdateProfile(userID, req.DisplayName)
	if err != nil {
		switch err.Error() {
		case "displayName is required", "displayName is too long":
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	response, err := toUserResponseDTO(c, user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(response)
}

func UploadAvatarHandler(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID format in token."})
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}

	user, err := UpdateAvatarWithUpload(context.Background(), userID, fileHeader)
	if err != nil {
		switch err.Error() {
		case "file is required":
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		default:
			if errors.Is(err, media.ErrAvatarFileTooLarge) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
			}
			if len(err.Error()) >= len("unsupported avatar file type:") && err.Error()[:len("unsupported avatar file type:")] == "unsupported avatar file type:" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	response, err := toUserResponseDTO(c, user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(response)
}

func UpdateGitHubAvatarHandler(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID format in token."})
	}

	var req UpdateGitHubAvatarRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body."})
	}

	user, err := UpdateAvatarWithGitHub(context.Background(), userID, req.Username)
	if err != nil {
		switch err.Error() {
		case "github username is required", "invalid github username":
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	response, err := toUserResponseDTO(c, user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(response)
}

func GetAvatarContentHandler(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing avatar token"})
	}

	tokenService, err := media.NewTokenService()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	claims, err := tokenService.VerifyAvatarReadToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid avatar token"})
	}

	user, err := GetUserByID(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}
	if trimStringPtr(user.AvatarObjectKey) == "" || trimStringPtr(user.AvatarObjectKey) != claims.ObjectKey {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Avatar token does not match current avatar"})
	}

	if err := media.InitStorageProviderForAvatarRead(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	obj, err := media.GetStoredObject(context.Background(), claims.ObjectKey)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}
	defer obj.Body.Close()

	c.Set("Content-Type", obj.ContentType)
	c.Set("Cache-Control", "private, max-age=60")
	data, err := io.ReadAll(obj.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read avatar content"})
	}
	return c.Send(data)
}

func getUserIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	userIdStr, ok := c.Locals("userId").(string)
	if !ok {
		return uuid.Nil, fiber.ErrUnauthorized
	}

	return uuid.Parse(userIdStr)
}

func toUserResponseDTO(c *fiber.Ctx, user *models.User) (UserResponseDTO, error) {
	avatarURL, err := ResolveAvatarURL(c.BaseURL(), user)
	if err != nil {
		return UserResponseDTO{}, err
	}
	return UserResponseDTO{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		AvatarURL:   avatarURL,
	}, nil
}
