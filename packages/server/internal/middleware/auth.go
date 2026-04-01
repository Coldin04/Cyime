package middleware

import (
	"os"
	"strings"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func jwtKeyFunc(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fiber.NewError(
			fiber.StatusUnauthorized,
			"unexpected signing method: "+token.Header["alg"].(string),
		)
	}
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		secret = "insecure-default-secret-for-dev-only"
	}
	return []byte(secret), nil
}

func parseJWTFromRequest(c *fiber.Ctx) (*auth.JWTClaims, error) {
	authHeader := c.Get("Authorization")
	tokenString := ""
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return nil, fiber.NewError(
				fiber.StatusUnauthorized,
				"Malformed Authorization header, expected 'Bearer {token}'",
			)
		}
		tokenString = parts[1]
	} else {
		tokenString = strings.TrimSpace(c.Cookies("cyime_media_access_token"))
		if tokenString == "" {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "Missing or malformed JWT")
		}
	}

	claims := &auth.JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, jwtKeyFunc)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid JWT")
	}
	return claims, nil
}

// Protected is a middleware that protects routes requiring a valid JWT.
// It verifies the token and passes the userId to the next handler via c.Locals().
func Protected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, err := parseJWTFromRequest(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Invalid or expired JWT",
				"details": err.Error(),
			})
		}
		c.Locals("userId", claims.UserID.String())
		return c.Next()
	}
}

// OptionalProtected parses JWT when provided, but does not block anonymous requests.
func OptionalProtected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		cookieToken := strings.TrimSpace(c.Cookies("cyime_media_access_token"))
		if strings.TrimSpace(authHeader) == "" && cookieToken == "" {
			return c.Next()
		}

		claims, err := parseJWTFromRequest(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Invalid or expired JWT",
				"details": err.Error(),
			})
		}
		c.Locals("userId", claims.UserID.String())
		return c.Next()
	}
}
