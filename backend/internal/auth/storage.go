package auth

// Storage contracts implemented by internal/pgstore, which is what the
// server runs on (see docs/adr/0001).

// UserStorage is the user-persistence contract used by handlers.
type UserStorage interface {
	Authenticate(login, password string) (*User, error)
	CreateUser(username, email, password, name string, roles []string, createdBy string) (*User, error)
	// CreateSignupUser is the self-service signup path: no roles are
	// granted yet, status is "pending" until ApproveUser is called.
	CreateSignupUser(username, email, password, name, requestedRole, createdBy string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id string) (*User, error)
	ListUsers() []*User
	// ListPendingUsers returns accounts awaiting superuser approval.
	ListPendingUsers() []*User
	// ApproveUser grants a pending account its requested role and flips it
	// to approved. RejectUser flips it to rejected without granting access.
	ApproveUser(id string) (*User, error)
	RejectUser(id string) (*User, error)
	// isActive is a pointer so a caller updating just the name doesn't
	// silently deactivate the account (Go's zero-value bool would write
	// is_active = false if this were a plain bool) — nil leaves it untouched.
	UpdateUser(id, name string, isActive *bool) (*User, error)
	UpdateUserRoles(id string, roles []string) (*User, error)
	GrantPermission(id string, p Permission) (*User, error)
	DenyPermission(id string, p Permission) (*User, error)
	RemovePermissionOverride(id string, p Permission, kind string) (*User, error)
	ResetPassword(id, newPassword string) error
	DeleteUser(id string) error
	IsRoleInUse(roleID string) bool
}

// RoleUsageChecker reports whether any user still has a role assigned.
type RoleUsageChecker interface {
	IsRoleInUse(roleID string) bool
}

// RoleStorage is the role-persistence contract used by handlers.
type RoleStorage interface {
	GetAll() []*Role
	GetByID(id string) (*Role, error)
	GetAsMap() map[string]*Role
	Create(r *Role) error
	Update(id string, updates *Role) error
	Delete(id string, usage RoleUsageChecker) error
}

// SessionStorage is the transcription-session persistence contract.
type SessionStorage interface {
	GetSessionsByUserID(userID string) []*SavedSession
	GetSessionByID(sessionID, userID string) (*SavedSession, error)
	SaveSession(session *SavedSession) (*SavedSession, error)
	UpdateSessionDocument(sessionID, userID string, document *SavedDocument) error
	DeleteSession(sessionID, userID string) error
}
