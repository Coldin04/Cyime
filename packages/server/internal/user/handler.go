package user

import (
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
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

// GetMe handles the GET /api/v1/user/me request.
// It relies on the Protected middleware to have already validated the JWT
// and placed the userId in the context.
func GetMe(c *fiber.Ctx) error {
	// Retrieve the userId string from the locals, which was set by the middleware.
	userIdStr, ok := c.Locals("userId").(string)
	if !ok {
		// This case should ideally not be reached if the middleware is correctly applied.
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: Invalid token context.",
		})
	}

	// Parse the string from the token's claim into a UUID.
	userID, err := uuid.Parse(userIdStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID format in token.",
		})
	}

	// Fetch the complete user record from the database.
	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found.",
		})
	}

	// Map the database model to our DTO (Data Transfer Object).
	response := UserResponseDTO{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
	}

	return c.JSON(response)
}
