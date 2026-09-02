package middleware

import (
	"github.com/gofiber/fiber/v2"
	"corti-backend/internal/auth"
)

// RequirePermission returns a Fiber middleware that rejects requests whose JWT
// claims do not include the specified permission.
// Must be placed after the auth middleware that populates c.Locals("claims").
func RequirePermission(p auth.Permission) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals("claims").(*auth.Claims)
		if !ok || claims == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}
		if !claims.HasPermission(p) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":    "permission denied",
				"required": p,
			})
		}
		return c.Next()
	}
}
