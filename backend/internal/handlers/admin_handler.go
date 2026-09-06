package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"corti-backend/internal/auth"
	"corti-backend/internal/pgstore"
)

// AdminHandler holds the superuser-only endpoints added for the signup-
// approval workflow: reviewing pending signups, and looking in on any
// user's saved sessions. All routes here require users.manage — the same
// permission that already gates /api/users and /api/roles, and one that
// admin/doctor accounts never have (see auth.DefaultRoles).
type AdminHandler struct {
	userStore    auth.UserStorage
	roleStore    auth.RoleStorage
	sessionStore auth.SessionStorage
	store        *pgstore.Store // demo_usage isn't part of the storage interfaces — see pgstore.Store.GetDemoUsage
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(userStore auth.UserStorage, roleStore auth.RoleStorage, sessionStore auth.SessionStorage, store *pgstore.Store) *AdminHandler {
	return &AdminHandler{
		userStore:    userStore,
		roleStore:    roleStore,
		sessionStore: sessionStore,
		store:        store,
	}
}

// HandleListPendingUsers returns every account awaiting approval.
// GET /api/users/pending  (requires users.manage)
func (h *AdminHandler) HandleListPendingUsers(c *fiber.Ctx) error {
	rolesMap := h.roleStore.GetAsMap()
	pending := h.userStore.ListPendingUsers()
	responses := make([]*auth.UserResponse, len(pending))
	for i, u := range pending {
		responses[i] = u.ToResponse(rolesMap)
	}
	return c.JSON(fiber.Map{
		"success": true,
		"users":   responses,
	})
}

// HandleApproveUser grants a pending account its requested role.
// POST /api/users/:id/approve  (requires users.manage)
func (h *AdminHandler) HandleApproveUser(c *fiber.Ctx) error {
	userID := c.Params("id")
	user, err := h.userStore.ApproveUser(userID)
	if err != nil {
		if err == auth.ErrUserNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": "User not found"})
		}
		log.Printf("Failed to approve user %s: %v", userID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to approve user"})
	}

	claims := c.Locals("claims").(*auth.Claims)
	log.Printf("User approved: %s roles=%v by %s", user.Email, user.Roles, claims.Email)

	return c.JSON(fiber.Map{
		"success": true,
		"user":    user.ToResponse(h.roleStore.GetAsMap()),
	})
}

// HandleRejectUser declines a pending account.
// POST /api/users/:id/reject  (requires users.manage)
func (h *AdminHandler) HandleRejectUser(c *fiber.Ctx) error {
	userID := c.Params("id")
	user, err := h.userStore.RejectUser(userID)
	if err != nil {
		if err == auth.ErrUserNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": "User not found"})
		}
		log.Printf("Failed to reject user %s: %v", userID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to reject user"})
	}

	claims := c.Locals("claims").(*auth.Claims)
	log.Printf("User rejected: %s by %s", user.Email, claims.Email)

	return c.JSON(fiber.Map{
		"success": true,
		"user":    user.ToResponse(h.roleStore.GetAsMap()),
	})
}

// HandleGetUserSessions lets a superuser view any user's saved sessions —
// "keep an eye on all the activity happening in the app," not just their
// own. Reuses the same SessionStorage.GetSessionsByUserID every user's own
// "My Sessions" screen already calls, just with an arbitrary target ID
// instead of always the caller's own.
// GET /api/admin/sessions/:userId  (requires users.manage)
func (h *AdminHandler) HandleGetUserSessions(c *fiber.Ctx) error {
	userID := c.Params("userId")
	if _, err := h.userStore.GetUserByID(userID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": "User not found"})
	}
	sessions := h.sessionStore.GetSessionsByUserID(userID)
	return c.JSON(fiber.Map{
		"success":  true,
		"sessions": sessions,
	})
}

// HandleGetAnalyticsOverview returns the org-wide stat-row counts for the
// Command Center dashboard.
// GET /api/admin/analytics/overview  (requires users.manage)
func (h *AdminHandler) HandleGetAnalyticsOverview(c *fiber.Ctx) error {
	overview, err := h.store.GetAnalyticsOverview(c.Context())
	if err != nil {
		log.Printf("Failed to load analytics overview: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to load analytics"})
	}
	return c.JSON(fiber.Map{"success": true, "overview": overview})
}

// HandleGetUsageTimeseries returns one usage point per day for the Command
// Center's usage chart.
// GET /api/admin/analytics/usage-timeseries?days=30  (requires users.manage)
func (h *AdminHandler) HandleGetUsageTimeseries(c *fiber.Ctx) error {
	days := c.QueryInt("days", 30)
	if days <= 0 || days > 365 {
		days = 30
	}
	series, err := h.store.GetUsageTimeseries(c.Context(), days)
	if err != nil {
		log.Printf("Failed to load usage timeseries: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to load usage timeseries"})
	}
	return c.JSON(fiber.Map{"success": true, "timeseries": series})
}

// HandleGetDemoFunnel returns every demo account with its lifetime
// per-feature trial usage, for the "trial account funnel" table.
// GET /api/admin/analytics/demo-funnel  (requires users.manage)
func (h *AdminHandler) HandleGetDemoFunnel(c *fiber.Ctx) error {
	funnel, err := h.store.GetDemoFunnel(c.Context())
	if err != nil {
		log.Printf("Failed to load demo funnel: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to load demo funnel"})
	}
	return c.JSON(fiber.Map{"success": true, "demoUsers": funnel})
}

// HandleGetRecentSessionsAllUsers returns the most recent sessions across
// every user, for the Command Center's cross-user activity feed. Distinct
// from HandleGetUserSessions, which scopes to one user.
// GET /api/admin/analytics/recent-sessions?limit=50  (requires users.manage)
func (h *AdminHandler) HandleGetRecentSessionsAllUsers(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	sessions, err := h.store.GetRecentSessions(c.Context(), limit)
	if err != nil {
		log.Printf("Failed to load recent sessions: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to load recent sessions"})
	}
	return c.JSON(fiber.Map{"success": true, "sessions": sessions})
}

// HandleGetMyDemoUsage returns the caller's own per-feature trial usage —
// only meaningful for a demo ("user" role) account, but available to
// anyone authenticated (a doctor/admin/superuser just gets empty/unused
// counts back, since middleware.RequireDemoAllowance never writes rows for
// them).
// GET /api/users/me/demo-usage
func (h *AdminHandler) HandleGetMyDemoUsage(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)
	usage, err := h.store.ListDemoUsage(c.Context(), claims.UserID)
	if err != nil {
		log.Printf("Failed to list demo usage for %s: %v", claims.UserID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to load trial usage"})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"usage":   usage,
	})
}
