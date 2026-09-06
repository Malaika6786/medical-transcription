package middleware

import (
	"log"
	"slices"

	"github.com/gofiber/fiber/v2"

	"corti-backend/internal/auth"
	"corti-backend/internal/pgstore"
)

// demoRoleID is the seeded role ID (auth.DefaultRoles) that marks an
// account as a trial/demo account subject to per-feature usage caps.
const demoRoleID = "user"

// demoTrialLimit is the lifetime (never resets) number of uses of a single
// feature a demo account gets before RequireDemoAllowance blocks it.
const demoTrialLimit = 3

// RequireDemoAllowance limits accounts with the demo ("user") role to
// demoTrialLimit lifetime uses of feature, then returns 403 with
// demoLimitReached:true so clients can show an upgrade message instead of a
// generic error. Doctor/admin/superuser accounts are never subject to this
// — it's a no-op for them. Placed on the specific action-creating route for
// a feature (not the whole route group), since some groups mix an action
// with reads/polling that shouldn't count as a "use".
func RequireDemoAllowance(store *pgstore.Store, feature string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals("claims").(*auth.Claims)
		if !ok || claims == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		if !slices.Contains(claims.Roles, demoRoleID) {
			return c.Next()
		}

		count, err := store.GetDemoUsage(c.Context(), claims.UserID, feature)
		if err != nil {
			log.Printf("demo limit check failed for user=%s feature=%s: %v", claims.UserID, feature, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to check trial usage"})
		}
		if count >= demoTrialLimit {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":            "You've used all 3 free trials for this feature. Contact us to upgrade your account.",
				"demoLimitReached": true,
				"feature":          feature,
			})
		}
		if err := store.IncrementDemoUsage(c.Context(), claims.UserID, feature); err != nil {
			log.Printf("demo usage increment failed for user=%s feature=%s: %v", claims.UserID, feature, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to record trial usage"})
		}
		return c.Next()
	}
}
