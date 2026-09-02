package handlers

import (
	"log"

	"corti-backend/internal/auth"

	"github.com/gofiber/fiber/v2"
)

// RolesHandler handles role and permission management endpoints.
type RolesHandler struct {
	roleStore auth.RoleStorage
	userStore auth.UserStorage
}

// NewRolesHandler creates a new roles handler.
func NewRolesHandler(roleStore auth.RoleStorage, userStore auth.UserStorage) *RolesHandler {
	return &RolesHandler{
		roleStore: roleStore,
		userStore: userStore,
	}
}

// HandleListRoles returns all roles.
// GET /api/roles
func (h *RolesHandler) HandleListRoles(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"roles":   h.roleStore.GetAll(),
	})
}

// HandleGetRole returns a single role by ID.
// GET /api/roles/:id
func (h *RolesHandler) HandleGetRole(c *fiber.Ctx) error {
	role, err := h.roleStore.GetByID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Role not found",
		})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"role":    role,
	})
}

// HandleCreateRole creates a new custom role.
// POST /api/roles
func (h *RolesHandler) HandleCreateRole(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)

	var req struct {
		Name        string           `json:"name"`
		Description string           `json:"description"`
		Permissions []auth.Permission `json:"permissions"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "name is required",
		})
	}

	role := &auth.Role{
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
	}
	if role.Permissions == nil {
		role.Permissions = []auth.Permission{}
	}

	if err := h.roleStore.Create(role); err != nil {
		if err == auth.ErrRoleExists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"success": false,
				"error":   "A role with this ID already exists",
			})
		}
		if err == auth.ErrInvalidPerm {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   err.Error(),
			})
		}
		log.Printf("Failed to create role: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to create role",
		})
	}

	log.Printf("Role created: %s by %s", role.Name, claims.Email)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"role":    role,
	})
}

// HandleUpdateRole updates an existing role's name, description, and permissions.
// PUT /api/roles/:id
func (h *RolesHandler) HandleUpdateRole(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)
	roleID := c.Params("id")

	var req struct {
		Name        string           `json:"name"`
		Description string           `json:"description"`
		Permissions []auth.Permission `json:"permissions"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	updates := &auth.Role{
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
	}
	if updates.Permissions == nil {
		updates.Permissions = []auth.Permission{}
	}

	if err := h.roleStore.Update(roleID, updates); err != nil {
		switch err {
		case auth.ErrRoleNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": "Role not found"})
		case auth.ErrInvalidPerm:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
		default:
			log.Printf("Failed to update role %s: %v", roleID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to update role"})
		}
	}

	role, _ := h.roleStore.GetByID(roleID)
	log.Printf("Role updated: %s by %s", roleID, claims.Email)

	return c.JSON(fiber.Map{
		"success": true,
		"role":    role,
	})
}

// HandleDeleteRole deletes a custom role. System roles cannot be deleted.
// DELETE /api/roles/:id
func (h *RolesHandler) HandleDeleteRole(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*auth.Claims)
	roleID := c.Params("id")

	if err := h.roleStore.Delete(roleID, h.userStore); err != nil {
		switch err {
		case auth.ErrRoleNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": "Role not found"})
		case auth.ErrSystemRole:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "System roles cannot be deleted"})
		case auth.ErrRoleInUse:
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"success": false, "error": "Role is assigned to one or more users. Reassign users before deleting."})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to delete role"})
		}
	}

	log.Printf("Role deleted: %s by %s", roleID, claims.Email)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Role deleted successfully",
	})
}

// HandleListPermissions returns all defined permission constants.
// GET /api/permissions
func (h *RolesHandler) HandleListPermissions(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success":     true,
		"permissions": auth.AllPermissions,
	})
}
