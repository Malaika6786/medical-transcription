package auth

import (
	"errors"
	"time"

	"corti-backend/internal/ai"
	"github.com/google/uuid"
)

// Sentinel errors returned by the storage implementations (pgstore) and
// checked by handlers for specific HTTP responses.
var (
	ErrSessionNotFound    = errors.New("session not found")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrUsernameExists     = errors.New("username already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrRoleNotFound       = errors.New("role not found")
	ErrRoleExists         = errors.New("role already exists")
	ErrSystemRole         = errors.New("system roles cannot be deleted")
	ErrRoleInUse          = errors.New("role is assigned to one or more users and cannot be deleted")
	ErrInvalidPerm        = errors.New("one or more permissions are not valid")
)

// DocumentSection represents a section in a generated document.
type DocumentSection struct {
	Key     string `json:"key,omitempty"`
	Name    string `json:"name"`
	Text    string `json:"text,omitempty"`
	Content string `json:"content,omitempty"`
}

// SavedDocument represents a generated clinical document.
type SavedDocument struct {
	ID           string            `json:"id"`
	TemplateKey  string            `json:"templateKey"`
	TemplateName string            `json:"templateName"`
	Sections     []DocumentSection `json:"sections"`
	HTMLContent  string            `json:"htmlContent,omitempty"`
	CreatedAt    string            `json:"createdAt"`
	UpdatedAt    string            `json:"updatedAt"`
}

// SavedSession represents a saved transcription session.
type SavedSession struct {
	ID            string               `json:"id"`
	UserID        string               `json:"userId"`
	Type          string               `json:"type"` // "ambient", "file-transcription", "dictation"
	Title         string               `json:"title"`
	Transcript    string               `json:"transcript"`
	Document      *SavedDocument       `json:"document,omitempty"`
	Extraction    *ai.ExtractionResult `json:"extraction,omitempty"`
	InteractionID string               `json:"interactionId,omitempty"`
	CreatedAt     string               `json:"createdAt"`
	UpdatedAt     string               `json:"updatedAt"`
}

// Role is a named group of permissions stored in roles.json.
type Role struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	IsSystem    bool         `json:"is_system"`
	CreatedAt   time.Time    `json:"created_at"`
	Permissions []Permission `json:"permissions"`
}

// DefaultRoles returns the built-in system roles seeded on a fresh store.
func DefaultRoles() []*Role {
	now := time.Now()
	return []*Role{
		{
			ID:          "superuser",
			Name:        "Superuser",
			Description: "Full system access",
			IsSystem:    true,
			CreatedAt:   now,
			Permissions: []Permission{
				PermAmbientAccess,
				PermFileTranscription,
				PermDictation,
				PermClinicalFacts,
				PermUsersManage,
				PermDocumentationView,
				PermEmbeddedAssistant,
				PermTemplatesManage,
				PermCortiSectionsView,
			},
		},
		{
			ID:          "doctor",
			Name:        "Doctor",
			Description: "Medical transcription access",
			IsSystem:    true,
			CreatedAt:   now,
			Permissions: []Permission{
				PermAmbientAccess,
				PermFileTranscription,
				PermDictation,
				PermClinicalFacts,
				PermDocumentationView,
				PermEmbeddedAssistant,
			},
		},
		{
			ID:          "admin",
			Name:        "Admin",
			Description: "Same access as Doctor",
			IsSystem:    true,
			CreatedAt:   now,
			Permissions: []Permission{
				PermAmbientAccess,
				PermFileTranscription,
				PermDictation,
				PermClinicalFacts,
				PermDocumentationView,
				PermEmbeddedAssistant,
			},
		},
		{
			ID:          "user",
			Name:        "User",
			Description: "Demo/trial access — every feature is reachable but rate-limited to 3 lifetime uses each (internal/middleware.RequireDemoAllowance)",
			IsSystem:    true,
			CreatedAt:   now,
			Permissions: []Permission{
				PermAmbientAccess,
				PermFileTranscription,
				PermDictation,
				PermClinicalFacts,
				PermDocumentationView,
				PermEmbeddedAssistant,
			},
		},
	}
}

// User represents a user in the system.
type User struct {
	ID           string       `json:"id"`
	Username     string       `json:"username"`
	Email        string       `json:"email"`
	PasswordHash string       `json:"passwordHash,omitempty"`
	Name         string       `json:"name"`
	Roles        []string     `json:"roles"`
	GrantedPerms []Permission `json:"granted_permissions"`
	DeniedPerms  []Permission `json:"denied_permissions"`
	IsActive     bool         `json:"isActive"`
	// Status is "pending" | "approved" | "rejected". Self-signups start
	// pending; admin-created accounts start approved (a superuser already
	// made that call). While not approved, EffectivePermissions and
	// GenerateToken both report zero permissions regardless of Roles — see
	// IsApproved.
	Status string `json:"status"`
	// RequestedRole holds the role a pending signup asked for, since Roles
	// stays empty until POST /users/:id/approve actually grants it.
	RequestedRole string    `json:"requestedRole,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	CreatedBy     string    `json:"createdBy,omitempty"`
	LastLogin     time.Time `json:"lastLogin,omitempty"`
}

// IsApproved reports whether this account's real roles/permissions should
// take effect. Checked by both GenerateToken (JWT claims) and ToResponse
// (the JSON both frontends read to decide what UI to show) so a pending or
// rejected account never sees or can exercise access it doesn't have yet,
// even though its Roles/GrantedPerms are left untouched in storage.
func (u *User) IsApproved() bool {
	return u.Status == "approved"
}

// EffectivePermissions computes the final permission set:
//
//	role permissions + user grants - user denies
//
// Returns empty for a pending/rejected account regardless of Roles — see
// IsApproved.
func (u *User) EffectivePermissions(rolesMap map[string]*Role) []Permission {
	if !u.IsApproved() {
		return []Permission{}
	}
	effective := make(map[Permission]bool)
	for _, id := range u.Roles {
		if r, ok := rolesMap[id]; ok {
			for _, p := range r.Permissions {
				effective[p] = true
			}
		}
	}
	for _, p := range u.GrantedPerms {
		effective[p] = true
	}
	for _, p := range u.DeniedPerms {
		delete(effective, p)
	}
	result := make([]Permission, 0, len(effective))
	for p := range effective {
		result = append(result, p)
	}
	return result
}

// HasPermission checks whether p is in the user's effective permission set.
func (u *User) HasPermission(p Permission, rolesMap map[string]*Role) bool {
	for _, ep := range u.EffectivePermissions(rolesMap) {
		if ep == p {
			return true
		}
	}
	return false
}

// LoginRequest represents login credentials. Login may be either the
// account's email or its username — Authenticate checks both.
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// SignupRequest represents a public self-service account creation request.
// The account is created pending — Role is only granted once a superuser
// approves it (POST /api/users/:id/approve). Role must be "user", "doctor",
// or "admin" — "superuser" can never be requested through signup.
type SignupRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

// LoginResponse represents a successful login response.
type LoginResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token"`
	User    *UserResponse `json:"user"`
}

// CreateUserRequest represents a request to create a new user.
type CreateUserRequest struct {
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Name     string   `json:"name"`
	Roles    []string `json:"roles"`
}

// UserResponse is the safe API representation of a user (no password hash).
type UserResponse struct {
	ID            string       `json:"id"`
	Username      string       `json:"username"`
	Email         string       `json:"email"`
	Name          string       `json:"name"`
	Roles         []string     `json:"roles"`
	Permissions   []Permission `json:"permissions"`
	GrantedPerms  []Permission `json:"granted_permissions"`
	DeniedPerms   []Permission `json:"denied_permissions"`
	IsActive      bool         `json:"isActive"`
	Status        string       `json:"status"`
	RequestedRole string       `json:"requestedRole,omitempty"`
	CreatedAt     time.Time    `json:"createdAt"`
	LastLogin     time.Time    `json:"lastLogin,omitempty"`
}

// PermissionBreakdown is returned by GET /api/users/:id/permissions.
type PermissionBreakdown struct {
	Effective  []Permission `json:"effective"`
	FromRoles  []Permission `json:"from_roles"`
	Granted    []Permission `json:"granted"`
	Denied     []Permission `json:"denied"`
}

// NewUser creates a new user with a generated (UUID) ID, status "approved"
// (the default for the admin-created-user path — CreateUserRequest has no
// approval concept). Postgres storage overwrites ID with a sequential value
// after calling this — see pgstore.Store.CreateUser. Self-signup
// (HandleSignup) builds its own User by hand instead, since it needs
// status "pending" and RequestedRole rather than these defaults.
func NewUser(username, email, passwordHash, name string, roles []string, createdBy string) *User {
	if roles == nil {
		roles = []string{}
	}
	return &User{
		ID:           uuid.New().String(),
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Name:         name,
		Roles:        roles,
		GrantedPerms: []Permission{},
		DeniedPerms:  []Permission{},
		IsActive:     true,
		Status:       "approved",
		CreatedAt:    time.Now(),
		CreatedBy:    createdBy,
	}
}

// ToResponse converts a User to a UserResponse using effective permissions from the given role map.
func (u *User) ToResponse(rolesMap map[string]*Role) *UserResponse {
	roles := u.Roles
	if roles == nil {
		roles = []string{}
	}
	granted := u.GrantedPerms
	if granted == nil {
		granted = []Permission{}
	}
	denied := u.DeniedPerms
	if denied == nil {
		denied = []Permission{}
	}
	return &UserResponse{
		ID:            u.ID,
		Username:      u.Username,
		Email:         u.Email,
		Name:          u.Name,
		Roles:         roles,
		Permissions:   u.EffectivePermissions(rolesMap),
		GrantedPerms:  granted,
		DeniedPerms:   denied,
		IsActive:      u.IsActive,
		Status:        u.Status,
		RequestedRole: u.RequestedRole,
		CreatedAt:     u.CreatedAt,
		LastLogin:     u.LastLogin,
	}
}
