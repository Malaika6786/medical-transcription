package handlers

import (
	"log"
	"strings"

	"corti-backend/internal/auth"

	"github.com/gofiber/fiber/v2"
)

// AuthHandler handles authentication and user-management endpoints.
type AuthHandler struct {
	userStore  auth.UserStorage
	roleStore  auth.RoleStorage
	jwtManager *auth.JWTManager
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(userStore auth.UserStorage, roleStore auth.RoleStorage, jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		userStore:  userStore,
		roleStore:  roleStore,
		jwtManager: jwtManager,
	}
}

// HandleLogin handles user login.
// POST /api/auth/login
func (h *AuthHandler) HandleLogin(c *fiber.Ctx) error {
	var req auth.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}
	if req.Login == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Username/email and password are required",
		})
	}

	user, err := h.userStore.Authenticate(req.Login, req.Password)
	if err != nil {
		log.Printf("Login failed for %s: %v", req.Login, err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid username/email or password",
		})
	}

	rolesMap := h.roleStore.GetAsMap()
	token, err := h.jwtManager.GenerateToken(user, rolesMap)
	if err != nil {
		log.Printf("Failed to generate token for %s: %v", req.Login, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to generate token",
		})
	}

	log.Printf("User logged in: %s roles=%v", user.Email, user.Roles)

	return c.JSON(auth.LoginResponse{
		Success: true,
		Token:   token,
		User:    user.ToResponse(rolesMap),
	})
}

// signupRoles are the only roles a self-signup may request. "superuser" is
// deliberately excluded — that account is provisioned once at seed time
// (admin@xstek.net) and never through this endpoint.
var signupRoles = map[string]bool{"user": true, "doctor": true, "admin": true}

// HandleSignup handles public self-service account creation. The account is
// created with status "pending" and no roles granted yet — RequestedRole
// records what they asked for, and a superuser must call
// POST /api/users/:id/approve before it does anything (see auth.User.IsApproved).
// POST /api/auth/signup
func (h *AuthHandler) HandleSignup(c *fiber.Ctx) error {
	var req auth.SignupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	req.Name = strings.TrimSpace(req.Name)
	req.Role = strings.TrimSpace(req.Role)

	if req.Username == "" || req.Email == "" || req.Password == "" || req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Username, email, name, and password are required",
		})
	}
	if !strings.Contains(req.Email, "@") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Enter a valid email address",
		})
	}
	if len(req.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Password must be at least 6 characters",
		})
	}
	// Backward-compat: a client that doesn't send a role yet (old web
	// frontend, before it's updated) defaults to the least-privileged role
	// rather than failing outright.
	if req.Role == "" {
		req.Role = "user"
	}
	if !signupRoles[req.Role] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Role must be one of: user, doctor, admin",
		})
	}

	user, err := h.userStore.CreateSignupUser(req.Username, req.Email, req.Password, req.Name, req.Role, "self-signup")
	if err != nil {
		switch err {
		case auth.ErrUserExists:
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"success": false,
				"error":   "An account with this email already exists",
			})
		case auth.ErrUsernameExists:
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"success": false,
				"error":   "This username is already taken",
			})
		default:
			log.Printf("Signup failed for %s: %v", req.Email, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "Failed to create account",
			})
		}
	}

	rolesMap := h.roleStore.GetAsMap()
	token, err := h.jwtManager.GenerateToken(user, rolesMap)
	if err != nil {
		log.Printf("Failed to generate token after signup for %s: %v", req.Email, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Account created, but automatic login failed — please sign in",
		})
	}

	log.Printf("New signup pending approval: %s (username=%s, id=%s, requestedRole=%s)", user.Email, user.Username, user.ID, user.RequestedRole)

	return c.Status(fiber.StatusCreated).JSON(auth.LoginResponse{
		Success: true,
		Token:   token,
		User:    user.ToResponse(rolesMap),
	})
}

// HandleGetCurrentUser returns the current authenticated user.
// GET /api/auth/me
func (h *AuthHandler) HandleGetCurrentUser(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)

	user, err := h.userStore.GetUserByID(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "User not found",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"user":    user.ToResponse(h.roleStore.GetAsMap()),
	})
}

// HandleListUsers returns all users. Route requires users.manage permission.
// GET /api/users
func (h *AuthHandler) HandleListUsers(c *fiber.Ctx) error {
	rolesMap := h.roleStore.GetAsMap()
	users := h.userStore.ListUsers()
	responses := make([]*auth.UserResponse, len(users))
	for i, u := range users {
		responses[i] = u.ToResponse(rolesMap)
	}
	return c.JSON(fiber.Map{
		"success": true,
		"users":   responses,
	})
}

// HandleCreateUser creates a new user. Route requires users.manage permission.
// POST /api/users
func (h *AuthHandler) HandleCreateUser(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)

	var req auth.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	if req.Username == "" || req.Email == "" || req.Password == "" || req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Username, email, password, and name are required",
		})
	}

	// Default to doctor role if none specified.
	if len(req.Roles) == 0 {
		req.Roles = []string{"doctor"}
	}

	// Validate that all requested roles exist.
	for _, roleID := range req.Roles {
		if _, err := h.roleStore.GetByID(roleID); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "Role not found: " + roleID,
			})
		}
	}

	user, err := h.userStore.CreateUser(req.Username, req.Email, req.Password, req.Name, req.Roles, claims.UserID)
	if err != nil {
		if err == auth.ErrUserExists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"success": false,
				"error":   "User with this email already exists",
			})
		}
		if err == auth.ErrUsernameExists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"success": false,
				"error":   "This username is already taken",
			})
		}
		log.Printf("Failed to create user: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to create user",
		})
	}

	log.Printf("User created: %s roles=%v by %s", user.Email, user.Roles, claims.Email)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"user":    user.ToResponse(h.roleStore.GetAsMap()),
	})
}

// HandleUpdateUser updates a user's name and active status. Route requires users.manage permission.
// PUT /api/users/:id
func (h *AuthHandler) HandleUpdateUser(c *fiber.Ctx) error {
	userID := c.Params("id")

	var req struct {
		Name     string `json:"name"`
		IsActive *bool  `json:"isActive"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	user, err := h.userStore.UpdateUser(userID, req.Name, req.IsActive)
	if err != nil {
		if err == auth.ErrUserNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to update user",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"user":    user.ToResponse(h.roleStore.GetAsMap()),
	})
}

// HandleDeleteUser deletes a user. Route requires users.manage permission.
// DELETE /api/users/:id
func (h *AuthHandler) HandleDeleteUser(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)
	userID := c.Params("id")

	if userID == claims.UserID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Cannot delete your own account",
		})
	}

	if err := h.userStore.DeleteUser(userID); err != nil {
		if err == auth.ErrUserNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to delete user",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User deleted successfully",
	})
}

// HandleResetPassword lets an admin set a new password for any user.
// PUT /api/users/:id/password  (requires users.manage)
func (h *AuthHandler) HandleResetPassword(c *fiber.Ctx) error {
	userID := c.Params("id")

	var req struct {
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}
	if len(req.NewPassword) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Password must be at least 6 characters",
		})
	}

	if err := h.userStore.ResetPassword(userID, req.NewPassword); err != nil {
		if err == auth.ErrUserNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "User not found",
			})
		}
		log.Printf("Failed to reset password for user %s: %v", userID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to reset password",
		})
	}

	claims := c.Locals("claims").(*auth.Claims)
	log.Printf("Password reset for user %s by %s", userID, claims.Email)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Password reset successfully",
	})
}

// HandleForceLogoutAll invalidates all user tokens. Route requires users.manage permission.
// POST /api/auth/force-logout-all
func (h *AuthHandler) HandleForceLogoutAll(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)

	if err := h.jwtManager.InvalidateAllTokens(); err != nil {
		log.Printf("Failed to invalidate tokens: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to invalidate tokens",
		})
	}

	log.Printf("Force logout triggered by %s - all tokens invalidated", claims.Email)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "All user sessions have been invalidated. Users will need to log in again.",
	})
}

// HandleGetUserPermissions returns effective permissions breakdown for a user.
// GET /api/users/:id/permissions  (requires users.manage)
func (h *AuthHandler) HandleGetUserPermissions(c *fiber.Ctx) error {
	userID := c.Params("id")
	user, err := h.userStore.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "User not found",
		})
	}

	rolesMap := h.roleStore.GetAsMap()

	// Collect permissions from roles only.
	fromRolesSet := make(map[auth.Permission]bool)
	for _, id := range user.Roles {
		if r, ok := rolesMap[id]; ok {
			for _, p := range r.Permissions {
				fromRolesSet[p] = true
			}
		}
	}
	fromRoles := permMapToSlice(fromRolesSet)

	breakdown := auth.PermissionBreakdown{
		Effective: user.EffectivePermissions(rolesMap),
		FromRoles: fromRoles,
		Granted:   nilSafe(user.GrantedPerms),
		Denied:    nilSafe(user.DeniedPerms),
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    breakdown,
	})
}

// HandleUpdateUserRoles replaces the role list for a user.
// PUT /api/users/:id/roles  (requires users.manage)
func (h *AuthHandler) HandleUpdateUserRoles(c *fiber.Ctx) error {
	userID := c.Params("id")

	var req struct {
		Roles []string `json:"roles"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Validate all roles exist.
	for _, roleID := range req.Roles {
		if _, err := h.roleStore.GetByID(roleID); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "Role not found: " + roleID,
			})
		}
	}

	user, err := h.userStore.UpdateUserRoles(userID, req.Roles)
	if err != nil {
		if err == auth.ErrUserNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to update roles",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"user":    user.ToResponse(h.roleStore.GetAsMap()),
	})
}

// HandleGrantPermission adds a user-level permission grant.
// POST /api/users/:id/permissions/grant  (requires users.manage)
func (h *AuthHandler) HandleGrantPermission(c *fiber.Ctx) error {
	userID := c.Params("id")

	var req struct {
		Permission auth.Permission `json:"permission"`
	}
	if err := c.BodyParser(&req); err != nil || req.Permission == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "permission field is required",
		})
	}

	user, err := h.userStore.GrantPermission(userID, req.Permission)
	if err != nil {
		if err == auth.ErrUserNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": "User not found"})
		}
		log.Printf("auth_handler: grant permission: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to grant permission"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"user":    user.ToResponse(h.roleStore.GetAsMap()),
	})
}

// HandleDenyPermission adds a user-level permission deny.
// POST /api/users/:id/permissions/deny  (requires users.manage)
func (h *AuthHandler) HandleDenyPermission(c *fiber.Ctx) error {
	userID := c.Params("id")

	var req struct {
		Permission auth.Permission `json:"permission"`
	}
	if err := c.BodyParser(&req); err != nil || req.Permission == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "permission field is required",
		})
	}

	user, err := h.userStore.DenyPermission(userID, req.Permission)
	if err != nil {
		if err == auth.ErrUserNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": "User not found"})
		}
		log.Printf("auth_handler: deny permission: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to deny permission"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"user":    user.ToResponse(h.roleStore.GetAsMap()),
	})
}

// HandleRemovePermissionOverride removes a grant or deny override for a user.
// DELETE /api/users/:id/permissions/:perm?type=grant|deny  (requires users.manage)
func (h *AuthHandler) HandleRemovePermissionOverride(c *fiber.Ctx) error {
	userID := c.Params("id")
	perm := auth.Permission(c.Params("perm"))
	kind := c.Query("type", "grant")

	user, err := h.userStore.RemovePermissionOverride(userID, perm, kind)
	if err != nil {
		if err == auth.ErrUserNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": "User not found"})
		}
		if err == auth.ErrInvalidOverrideKind {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
		}
		log.Printf("auth_handler: remove permission override: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to remove permission override"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"user":    user.ToResponse(h.roleStore.GetAsMap()),
	})
}

// AuthMiddleware validates the JWT token and stores claims in the request context.
func (h *AuthHandler) AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Authorization header required",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid authorization header format",
			})
		}

		claims, err := h.jwtManager.ValidateToken(parts[1])
		if err != nil {
			if err == auth.ErrExpiredToken {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"success": false,
					"error":   "Token expired",
				})
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid token",
			})
		}

		c.Locals("claims", claims)
		return c.Next()
	}
}

// AuthMiddlewareAllowQueryToken is AuthMiddleware's counterpart for
// WebSocket routes that need real authentication but can't rely on an
// Authorization header — browsers don't send custom headers during a WS
// handshake. It accepts the token either as a normal `Bearer <token>`
// header (mobile clients that support it) or as a `?token=` query
// parameter (the fallback every client, including the browser, can use) —
// the same "token in the URL" approach this backend already uses for its
// own outbound Corti WebSocket connections (see corti/ambient_proxy.go).
//
// Use this instead of AuthMiddleware only where a header genuinely can't be
// sent; prefer the header-based AuthMiddleware everywhere else, since a URL
// (query string) is more likely to be logged somewhere than a header.
func (h *AuthHandler) AuthMiddlewareAllowQueryToken() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenStr := ""
		if authHeader := c.Get("Authorization"); authHeader != "" {
			if parts := strings.Split(authHeader, " "); len(parts) == 2 && parts[0] == "Bearer" {
				tokenStr = parts[1]
			}
		}
		if tokenStr == "" {
			tokenStr = c.Query("token")
		}
		if tokenStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Authorization required (Bearer header or ?token= query parameter)",
			})
		}

		claims, err := h.jwtManager.ValidateToken(tokenStr)
		if err != nil {
			if err == auth.ErrExpiredToken {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "error": "Token expired"})
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "error": "Invalid token"})
		}

		c.Locals("claims", claims)
		return c.Next()
	}
}

// --- helpers ---

func permMapToSlice(m map[auth.Permission]bool) []auth.Permission {
	out := make([]auth.Permission, 0, len(m))
	for p := range m {
		out = append(out, p)
	}
	return out
}

func nilSafe(perms []auth.Permission) []auth.Permission {
	if perms == nil {
		return []auth.Permission{}
	}
	return perms
}
