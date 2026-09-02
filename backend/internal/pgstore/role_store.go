package pgstore

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/google/uuid"

	"corti-backend/internal/auth"
)

const roleColumns = `id, name, description, is_system, created_at, permissions`

func scanRole(row pgx.Row) (*auth.Role, error) {
	var (
		r     auth.Role
		perms []string
	)
	err := row.Scan(&r.ID, &r.Name, &r.Description, &r.IsSystem, &r.CreatedAt, &perms)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, auth.ErrRoleNotFound
	}
	if err != nil {
		return nil, err
	}
	r.Permissions = stringsToPerms(perms)
	return &r, nil
}

// GetAll returns all roles.
func (s *Store) GetAll() []*auth.Role {
	rows, err := s.pool.Query(context.Background(), `SELECT `+roleColumns+` FROM roles ORDER BY name`)
	if err != nil {
		log.Printf("pgstore: list roles: %v", err)
		return []*auth.Role{}
	}
	defer rows.Close()
	roles := []*auth.Role{}
	for rows.Next() {
		r, err := scanRole(rows)
		if err != nil {
			log.Printf("pgstore: scan role: %v", err)
			continue
		}
		roles = append(roles, r)
	}
	return roles
}

// GetByID returns the role with the given ID, or ErrRoleNotFound.
func (s *Store) GetByID(id string) (*auth.Role, error) {
	row := s.pool.QueryRow(context.Background(), `SELECT `+roleColumns+` FROM roles WHERE id = $1`, id)
	return scanRole(row)
}

// GetAsMap returns all roles keyed by ID.
func (s *Store) GetAsMap() map[string]*auth.Role {
	m := make(map[string]*auth.Role)
	for _, r := range s.GetAll() {
		m[r.ID] = r
	}
	return m
}

// Create adds a new role.
func (s *Store) Create(r *auth.Role) error {
	if err := auth.ValidatePermissions(r.Permissions); err != nil {
		return err
	}
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	r.CreatedAt = time.Now()
	r.IsSystem = false
	_, err := s.pool.Exec(context.Background(), `
		INSERT INTO roles (id, name, description, is_system, created_at, permissions)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		r.ID, r.Name, r.Description, r.IsSystem, r.CreatedAt, permsToStrings(r.Permissions))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return auth.ErrRoleExists
		}
		return err
	}
	return nil
}

// UpsertRole inserts or replaces a full role record (used by seeding and cmd/migrate-json).
func (s *Store) UpsertRole(ctx context.Context, r *auth.Role) error {
	createdAt := r.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO roles (id, name, description, is_system, created_at, permissions)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			is_system = EXCLUDED.is_system,
			permissions = EXCLUDED.permissions`,
		r.ID, r.Name, r.Description, r.IsSystem, createdAt, permsToStrings(r.Permissions))
	return err
}

// Update replaces the name/description/permissions of an existing role.
func (s *Store) Update(id string, updates *auth.Role) error {
	if err := auth.ValidatePermissions(updates.Permissions); err != nil {
		return err
	}
	tag, err := s.pool.Exec(context.Background(), `
		UPDATE roles SET name = $2, description = $3, permissions = $4 WHERE id = $1`,
		id, updates.Name, updates.Description, permsToStrings(updates.Permissions))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return auth.ErrRoleNotFound
	}
	return nil
}

// Delete removes a role by ID. System roles and roles still assigned to users
// are protected.
func (s *Store) Delete(id string, usage auth.RoleUsageChecker) error {
	role, err := s.GetByID(id)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return auth.ErrSystemRole
	}
	if usage != nil && usage.IsRoleInUse(id) {
		return auth.ErrRoleInUse
	}
	_, err = s.pool.Exec(context.Background(), `DELETE FROM roles WHERE id = $1`, id)
	return err
}
