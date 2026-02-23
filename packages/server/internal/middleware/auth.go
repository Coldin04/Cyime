package middleware

import (
	"os"
	"strings"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// Protected is a middleware that protects routes requiring a valid JWT.
// It verifies the token and passes the userId to the next handler via c.Locals().
func Protected() fiber.Handler {
	// This is the key function for the JWT library to get the secret.
	// It's defined once here to avoid repetition.
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		// Validate the alg is what we expect: HS256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "unexpected signing method: "+token.Header["alg"].(string))
		}
		
		// Fetch the secret key. This logic should be consistent with how the token was signed.
		secret := os.Getenv("JWT_SECRET_KEY")
		if secret == "" {
			secret = "insecure-default-secret-for-dev-only"
		}
		return []byte(secret), nil
	}

	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing or malformed JWT",
			})
		}

		// The header should be in the format "Bearer {token}"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Malformed Authorization header, expected 'Bearer {token}'",
			})
		}
		tokenString := parts[1]

		// Parse and validate the token.
		claims := &auth.JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, keyFunc)

		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired JWT",
				"details": err.Error(),
			})
		}

		if !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid JWT",
			})
		}

		// Store the userId in locals for the next handler to use.
		// The key "userId" must be consistently used by handlers.
		c.Locals("userId", claims.UserID.String())

		return c.Next()
	}
}
