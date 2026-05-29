package middleware

import (
	"time"

	httpauth "project_smt6/app/http/auth"
	"project_smt6/app/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func Auth(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := httpauth.BearerToken(c)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication token is required"})
		}

		var session models.Session
		err := db.Preload("User").
			Where("token_hash = ? AND expired_at > ?", httpauth.TokenHash(token), time.Now()).
			First(&session).Error
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication token is invalid or expired"})
		}

		c.Locals("session", session)
		c.Locals("user", session.User)
		return c.Next()
	}
}

func Role(roles ...models.UserRole) fiber.Handler {
	allowed := make(map[models.UserRole]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		user, ok := c.Locals("user").(models.User)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthenticated"})
		}
		if _, ok := allowed[user.Role]; !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "insufficient permission"})
		}
		return c.Next()
	}
}
