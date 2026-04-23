package middleware

import (
	"livo-fiber-backend/database"
	"livo-fiber-backend/models"
	"math"

	"github.com/gofiber/fiber/v3"
)

// RoleMiddleware checks if user has required role hierarchy
func RoleMiddleware(allowedRoles []string) fiber.Handler {
	return func(c fiber.Ctx) error {
		userRoles, ok := c.Locals("userRoles").([]string)
		if !ok || len(userRoles) == 0 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient permissions",
			})
		}

		// Get maximum hierarchy level from allowed roles.
		// Lower numbers mean higher privilege, so users with hierarchy <= maxAllowedHierarchy are permitted.
		maxAllowedHierarchy := math.MinInt
		for _, allowedRole := range allowedRoles {
			var role models.Role
			if err := database.DB.Where("role_name = ?", allowedRole).First(&role).Error; err == nil {
				if role.Hierarchy > maxAllowedHierarchy {
					maxAllowedHierarchy = role.Hierarchy
				}
			}
		}

		if maxAllowedHierarchy == math.MinInt {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient permissions",
			})
		}

		// Check if user has any role with equal or higher privilege
		for _, userRole := range userRoles {
			var role models.Role
			if err := database.DB.Where("role_name = ?", userRole).First(&role).Error; err == nil {
				if role.Hierarchy <= maxAllowedHierarchy {
					return c.Next()
				}
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}
}
