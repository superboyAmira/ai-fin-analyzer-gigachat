package middleware

import (
	"rag-iishka/pkg/auth"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func AuthMiddleware(jwtManager *auth.JWTManager, logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Get("Authorization")
		if token == "" {
			logger.Warn("Missing authorization token")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization token required",
			})
		}

		// Remove "Bearer " prefix if present
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			logger.Warn("Invalid token", zap.Error(err))
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// Store claims in context
		c.Locals("userID", claims.UserID)
		c.Locals("username", claims.Username)
		c.Locals("email", claims.Email)

		return c.Next()
	}
}

