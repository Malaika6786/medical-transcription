package auth

// Storage contracts implemented by both the JSON-file stores (kept for tests
// and as the source read by cmd/migrate-json) and the Postgres stores in
// internal/pgstore, which are what the server runs on (see docs/adr/0001).

// UserStorage is the user-persistence contract used by handlers.
type UserStorage interface {
	Authenticate(login, password string) (*User, error)
	CreateUser(username, email, password, name string, roles []string, createdBy string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id string) (*User, error)
	ListUsers() []*User
	UpdateUser(id, name string, isActive bool) (*User, error)
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
