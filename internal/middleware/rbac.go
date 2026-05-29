package middleware

import (
	"network-monitor/internal/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var rolePermissions = map[string][]string{
	"atasan": {
		"users:read", "users:write", "users:delete",
		"roles:read", "roles:write",
		"monitoring:read", "monitoring:write",
		"devices:read", "devices:write", "devices:delete",
		"feedback:read", "feedback:write", "feedback:respond", "feedback:delete",
	},
	"teknisi_it": {
		"users:read",
		"monitoring:read", "monitoring:write",
		"devices:read", "devices:write",
		"feedback:read", "feedback:respond",
	},
	"staff": {
		"monitoring:read",
		"devices:read",
		"feedback:read", "feedback:write",
	},
	"karyawan": {
		"feedback:write",
	},
}

func RequirePermission(db *gorm.DB, permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := GetRole(c)
		if role == "" {
			return c.Status(403).JSON(fiber.Map{
				"success": false,
				"message": "Akses ditolak: role tidak ditemukan.",
			})
		}

		perms, exists := rolePermissions[role]
		if !exists {
			var dbRole models.Role
			if err := db.Where("name = ?", role).Preload("Permissions").First(&dbRole).Error; err == nil {
				for _, p := range dbRole.Permissions {
					if p.Name == permission {
						return c.Next()
					}
				}
			}
			return c.Status(403).JSON(fiber.Map{
				"success": false,
				"message": "Akses ditolak: role tidak memiliki izin ini.",
			})
		}

		for _, p := range perms {
			if p == permission {
				return c.Next()
			}
		}

		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"message": "Akses ditolak: Anda tidak memiliki izin untuk aksi ini.",
		})
	}
}

func RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := GetRole(c)
		for _, r := range roles {
			if r == userRole {
				return c.Next()
			}
		}
		return c.Status(403).JSON(fiber.Map{
			"success": false,
			"message": "Akses ditolak: role Anda tidak diizinkan mengakses resource ini.",
		})
	}
}

func HasPermission(role, permission string) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}
