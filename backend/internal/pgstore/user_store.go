package pgstore

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"corti-backend/internal/auth"
)

const userColumns = `id, username, email, password_hash, name, roles, granted_permissions, denied_permissions, is_active, status, requested_role, created_at, created_by, last_login`

const uniqueViolation = "23505"

func scanUser(row pgx.Row) (*auth.User, error) {
	var (
		u               auth.User
		granted, denied []string
		lastLogin       *time.Time
	)
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Name, &u.Roles, &granted, &denied, &u.IsActive, &u.Status, &u.RequestedRole, &u.CreatedAt, &u.CreatedBy, &lastLogin)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, auth.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	u.GrantedPerms = stringsToPerms(granted)
	u.DeniedPerms = stringsToPerms(denied)
	if lastLogin != nil {
		u.LastLogin = *lastLogin
	}
	return &u, nil
}

func (s *Store) getUserBy(ctx context.Context, where string, arg any) (*auth.User, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE `+where, arg)
	return scanUser(row)
}

// Authenticate validates credentials and returns the user. login may be
// either the account's email or its username.
func (s *Store) Authenticate(login, password string) (*auth.User, error) {
	ctx := context.Background()
	user, err := s.getUserBy(ctx, `email = $1 OR username = $1`, login)
	if err != nil || !user.IsActive {
		return nil, auth.ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, auth.ErrInvalidCredentials
	}
	now := time.Now()
	user.LastLogin = now
	if _, err := s.pool.Exec(ctx, `UPDATE users SET last_login = $2 WHERE id = $1`, user.ID, now); err != nil {
		log.Printf("pgstore: update last_login: %v", err)
	}
	return user, nil
}

// CreateUser creates a new user with the given roles. The ID is a
// sequential, zero-padded string ("00", "01", ...) assigned from
// user_id_seq, not the UUID auth.NewUser generates by default.
func (s *Store) CreateUser(username, email, password, name string, roles []string, createdBy string) (*auth.User, error) {
	ctx := context.Background()
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	id, err := s.nextUserID(ctx)
	if err != nil {
		return nil, err
	}
	user := auth.NewUser(username, email, string(hashed), name, roles, createdBy)
	user.ID = id
	_, err = s.pool.Exec(ctx, `
		INSERT INTO users (id, username, email, password_hash, name, roles, granted_permissions, denied_permissions, is_active, status, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, '{}', '{}', TRUE, 'approved', $7, $8)`,
		user.ID, user.Username, user.Email, user.PasswordHash, user.Name, user.Roles, user.CreatedAt, user.CreatedBy)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			if pgErr.ConstraintName == "users_username_key" {
				return nil, auth.ErrUsernameExists
			}
			return nil, auth.ErrUserExists
		}
		return nil, err
	}
	return user, nil
}

// CreateSignupUser creates a new self-service signup: no roles are granted
// yet (requestedRole records what they asked for) and status is "pending"
// until a superuser calls ApproveUser. See auth.User.IsApproved for how
// this is enforced even though the account otherwise exists and can log in.
func (s *Store) CreateSignupUser(username, email, password, name, requestedRole, createdBy string) (*auth.User, error) {
	ctx := context.Background()
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	id, err := s.nextUserID(ctx)
	if err != nil {
		return nil, err
	}
	user := auth.NewUser(username, email, string(hashed), name, []string{}, createdBy)
	user.ID = id
	user.Status = "pending"
	user.RequestedRole = requestedRole
	_, err = s.pool.Exec(ctx, `
		INSERT INTO users (id, username, email, password_hash, name, roles, granted_permissions, denied_permissions, is_active, status, requested_role, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, '{}', '{}', '{}', TRUE, 'pending', $6, $7, $8)`,
		user.ID, user.Username, user.Email, user.PasswordHash, user.Name, user.RequestedRole, user.CreatedAt, user.CreatedBy)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			if pgErr.ConstraintName == "users_username_key" {
				return nil, auth.ErrUsernameExists
			}
			return nil, auth.ErrUserExists
		}
		return nil, err
	}
	return user, nil
}

// ListPendingUsers returns all users awaiting superuser approval, oldest
// request first.
func (s *Store) ListPendingUsers() []*auth.User {
	rows, err := s.pool.Query(context.Background(),
		`SELECT `+userColumns+` FROM users WHERE status = 'pending' ORDER BY created_at`)
	if err != nil {
		log.Printf("pgstore: list pending users: %v", err)
		return []*auth.User{}
	}
	defer rows.Close()
	users := []*auth.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			log.Printf("pgstore: scan pending user: %v", err)
			continue
		}
		users = append(users, u)
	}
	return users
}

// ApproveUser marks a pending (or rejected) account approved. If the
// account has no roles yet (the normal case for a fresh signup — see
// CreateSignupUser), roleIDForApprove resolves RequestedRole to an actual
// role ID and grants it; a retroactively-pending existing account already
// has real roles (see cmd/backfill-approval) and just needs the status flip.
func (s *Store) ApproveUser(id string) (*auth.User, error) {
	user, err := s.GetUserByID(id)
	if err != nil {
		return nil, err
	}
	if len(user.Roles) == 0 && user.RequestedRole != "" {
		if err := s.mustAffectUser(`UPDATE users SET status = 'approved', roles = $2 WHERE id = $1`,
			id, []string{user.RequestedRole}); err != nil {
			return nil, err
		}
	} else {
		if err := s.mustAffectUser(`UPDATE users SET status = 'approved' WHERE id = $1`, id); err != nil {
			return nil, err
		}
	}
	return s.GetUserByID(id)
}

// RejectUser marks a pending account rejected. Its (empty, for a fresh
// signup) roles are left as-is — rejecting never grants access.
func (s *Store) RejectUser(id string) (*auth.User, error) {
	if err := s.mustAffectUser(`UPDATE users SET status = 'rejected' WHERE id = $1`, id); err != nil {
		return nil, err
	}
	return s.GetUserByID(id)
}

// GetDemoUsage returns how many times userID has used feature (0 if never).
func (s *Store) GetDemoUsage(ctx context.Context, userID, feature string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT use_count FROM demo_usage WHERE user_id = $1 AND feature = $2`, userID, feature).Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return count, err
}

// IncrementDemoUsage records one more use of feature by userID.
func (s *Store) IncrementDemoUsage(ctx context.Context, userID, feature string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO demo_usage (user_id, feature, use_count) VALUES ($1, $2, 1)
		ON CONFLICT (user_id, feature) DO UPDATE SET use_count = demo_usage.use_count + 1`,
		userID, feature)
	return err
}

// BackfillPendingApproval retroactively marks every existing account except
// the hardcoded superuser as "pending" — see cmd/backfill-approval, the
// one-time manual tool that calls this. Real `roles` are left untouched;
// only status/requested_role change, so approving later is a pure status
// flip, never a re-grant.
func (s *Store) BackfillPendingApproval(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE users
		SET status = 'pending', requested_role = COALESCE(roles[1], 'user')
		WHERE email != 'admin@xstek.net' AND status = 'approved'`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ListDemoUsage returns feature -> use_count for every feature userID has
// touched at least once (a feature with no row simply means 0 uses).
func (s *Store) ListDemoUsage(ctx context.Context, userID string) (map[string]int, error) {
	rows, err := s.pool.Query(ctx, `SELECT feature, use_count FROM demo_usage WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	usage := map[string]int{}
	for rows.Next() {
		var feature string
		var count int
		if err := rows.Scan(&feature, &count); err != nil {
			return nil, err
		}
		usage[feature] = count
	}
	return usage, rows.Err()
}

// nextUserID draws the next value from user_id_seq and formats it as a
// zero-padded string, starting at "00".
func (s *Store) nextUserID(ctx context.Context) (string, error) {
	var n int64
	if err := s.pool.QueryRow(ctx, `SELECT nextval('user_id_seq')`).Scan(&n); err != nil {
		return "", fmt.Errorf("draw user id: %w", err)
	}
	return fmt.Sprintf("%02d", n-1), nil
}

// UpsertUser inserts or replaces a full user record (used by seeding and cmd/migrate-json).
func (s *Store) UpsertUser(ctx context.Context, u *auth.User) error {
	var lastLogin *time.Time
	if !u.LastLogin.IsZero() {
		ll := u.LastLogin
		lastLogin = &ll
	}
	createdAt := u.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	username := u.Username
	if username == "" {
		// Legacy JSON records predate the username column.
		username = strings.SplitN(u.Email, "@", 2)[0]
	}
	status := u.Status
	if status == "" {
		// Seeding and cmd/migrate-json both predate the approval workflow —
		// these are already-vetted accounts, not new self-signups.
		status = "approved"
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO users (id, username, email, password_hash, name, roles, granted_permissions, denied_permissions, is_active, status, requested_role, created_at, created_by, last_login)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			username = EXCLUDED.username,
			email = EXCLUDED.email,
			password_hash = EXCLUDED.password_hash,
			name = EXCLUDED.name,
			roles = EXCLUDED.roles,
			granted_permissions = EXCLUDED.granted_permissions,
			denied_permissions = EXCLUDED.denied_permissions,
			is_active = EXCLUDED.is_active,
			created_by = EXCLUDED.created_by,
			last_login = EXCLUDED.last_login`,
		u.ID, username, u.Email, u.PasswordHash, u.Name, u.Roles,
		permsToStrings(u.GrantedPerms), permsToStrings(u.DeniedPerms),
		u.IsActive, status, u.RequestedRole, createdAt, u.CreatedBy, lastLogin)
	return err
}

// GetUserByEmail returns a user by email.
func (s *Store) GetUserByEmail(email string) (*auth.User, error) {
	return s.getUserBy(context.Background(), `email = $1`, email)
}

// GetUserByID returns a user by ID.
func (s *Store) GetUserByID(id string) (*auth.User, error) {
	return s.getUserBy(context.Background(), `id = $1`, id)
}

// ListUsers returns all users.
func (s *Store) ListUsers() []*auth.User {
	rows, err := s.pool.Query(context.Background(), `SELECT `+userColumns+` FROM users ORDER BY created_at`)
	if err != nil {
		log.Printf("pgstore: list users: %v", err)
		return []*auth.User{}
	}
	defer rows.Close()
	users := []*auth.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			log.Printf("pgstore: scan user: %v", err)
			continue
		}
		users = append(users, u)
	}
	return users
}

// UpdateUser updates a user's name, and their active status only when
// isActive is non-nil — a name-only update must never flip is_active.
func (s *Store) UpdateUser(id, name string, isActive *bool) (*auth.User, error) {
	if isActive == nil {
		if err := s.mustAffectUser(`UPDATE users SET name = $2 WHERE id = $1`, id, name); err != nil {
			return nil, err
		}
	} else {
		if err := s.mustAffectUser(`UPDATE users SET name = $2, is_active = $3 WHERE id = $1`, id, name, *isActive); err != nil {
			return nil, err
		}
	}
	return s.GetUserByID(id)
}

// UpdateUserRoles replaces the role list for a user.
func (s *Store) UpdateUserRoles(id string, roles []string) (*auth.User, error) {
	if roles == nil {
		roles = []string{}
	}
	if err := s.mustAffectUser(`UPDATE users SET roles = $2 WHERE id = $1`, id, roles); err != nil {
		return nil, err
	}
	return s.GetUserByID(id)
}

// GrantPermission adds p to the user's granted overrides (a grant cancels a deny).
func (s *Store) GrantPermission(id string, p auth.Permission) (*auth.User, error) {
	return s.applyPermOverride(id, p, "granted_permissions", "denied_permissions")
}

// DenyPermission adds p to the user's denied overrides (a deny cancels a grant).
func (s *Store) DenyPermission(id string, p auth.Permission) (*auth.User, error) {
	return s.applyPermOverride(id, p, "denied_permissions", "granted_permissions")
}

// applyPermOverride appends p to target (deduplicated) and removes it from opposite.
func (s *Store) applyPermOverride(id string, p auth.Permission, target, opposite string) (*auth.User, error) {
	query := fmt.Sprintf(`
		UPDATE users SET
			%[1]s = CASE WHEN $2 = ANY(%[1]s) THEN %[1]s ELSE %[1]s || $2 END,
			%[2]s = array_remove(%[2]s, $2)
		WHERE id = $1`, target, opposite)
	if err := s.mustAffectUser(query, id, string(p)); err != nil {
		return nil, err
	}
	return s.GetUserByID(id)
}

// RemovePermissionOverride removes p from either the granted or denied list.
func (s *Store) RemovePermissionOverride(id string, p auth.Permission, kind string) (*auth.User, error) {
	var column string
	switch kind {
	case "grant":
		column = "granted_permissions"
	case "deny":
		column = "denied_permissions"
	default:
		return nil, auth.ErrInvalidOverrideKind
	}
	query := fmt.Sprintf(`UPDATE users SET %s = array_remove(%s, $2) WHERE id = $1`, column, column)
	if err := s.mustAffectUser(query, id, string(p)); err != nil {
		return nil, err
	}
	return s.GetUserByID(id)
}

// ResetPassword replaces a user's password hash with a new bcrypt hash.
func (s *Store) ResetPassword(id, newPassword string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.mustAffectUser(`UPDATE users SET password_hash = $2 WHERE id = $1`, id, string(hashed))
}

// DeleteUser removes a user by ID (their sessions cascade).
func (s *Store) DeleteUser(id string) error {
	return s.mustAffectUser(`DELETE FROM users WHERE id = $1`, id)
}

// IsRoleInUse returns true if any user has roleID assigned.
func (s *Store) IsRoleInUse(roleID string) bool {
	var inUse bool
	err := s.pool.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM users WHERE $1 = ANY(roles))`, roleID).Scan(&inUse)
	if err != nil {
		log.Printf("pgstore: role-in-use check: %v", err)
		return false
	}
	return inUse
}

// mustAffectUser executes a statement that must touch an existing user row,
// mapping zero affected rows to ErrUserNotFound.
func (s *Store) mustAffectUser(query string, args ...any) error {
	tag, err := s.pool.Exec(context.Background(), query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return auth.ErrUserNotFound
	}
	return nil
}
