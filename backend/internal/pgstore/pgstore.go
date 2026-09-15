// Package pgstore implements the auth storage contracts on PostgreSQL with
// the pgvector extension — the system of record for users, roles, sessions,
// Corti templates, and all embeddings (docs/adr/0001). SQL is hand-written on
// pgx; vectors travel as text literals cast to ::vector (docs/adr/0003).
package pgstore

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"corti-backend/internal/auth"
	"corti-backend/internal/cryptofield"
)

//go:embed schema.sql
var schemaSQL string

// ErrTemplateNotFound is returned when a Corti template key does not exist.
var ErrTemplateNotFound = errors.New("template not found")

// Store implements auth.UserStorage, auth.RoleStorage, and auth.SessionStorage
// on a single pgx connection pool, plus template, embedding, and search
// persistence.
type Store struct {
	pool *pgxpool.Pool
	// cipher encrypts/decrypts patient PII columns (nhs_number_enc,
	// name_enc, date_of_birth_enc — see patient_store.go). Nil until
	// SetFieldCipher is called; every patient_store.go method panics via
	// requireCipher if used before that, rather than silently writing
	// plaintext.
	cipher *cryptofield.Cipher
}

// SetFieldCipher wires in the encryption used for patient PII columns.
// Called once from main.go at startup, after loading FIELD_ENCRYPTION_KEY —
// kept as a separate setter (rather than a Connect parameter) so every
// existing pgstore.Connect call site is unaffected.
func (s *Store) SetFieldCipher(c *cryptofield.Cipher) {
	s.cipher = c
}

// newRandomID returns a 16-byte random hex string, used for ID prefixes
// that aren't drawn from the users table's sequential ID scheme (e.g.
// "pt_" + newRandomID() for patients — see patient_store.go).
func newRandomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failing is effectively unrecoverable (no entropy
		// source) — panicking here matches how the rest of the codebase
		// treats "this should never happen" conditions rather than
		// threading an error return through every ID-generating call site.
		panic(fmt.Sprintf("pgstore: crypto/rand failed: %v", err))
	}
	return hex.EncodeToString(b)
}

var (
	_ auth.UserStorage    = (*Store)(nil)
	_ auth.RoleStorage    = (*Store)(nil)
	_ auth.SessionStorage = (*Store)(nil)
)

// Connect opens a pool against databaseURL and verifies it with a ping.
func Connect(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to Postgres at %s: %w", databaseURL, err)
	}
	return &Store{pool: pool}, nil
}

// ApplySchema runs the embedded idempotent schema.
func (s *Store) ApplySchema(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

// EnsureSeed creates the default system roles and superuser on a fresh
// database, mirroring the JSON stores' first-run behaviour.
func (s *Store) EnsureSeed(ctx context.Context) error {
	var roleCount int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM roles`).Scan(&roleCount); err != nil {
		return err
	}
	if roleCount == 0 {
		for _, r := range auth.DefaultRoles() {
			if err := s.UpsertRole(ctx, r); err != nil {
				return fmt.Errorf("seed role %s: %w", r.ID, err)
			}
		}
		log.Println("pgstore: seeded default system roles")
	}

	var userCount int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&userCount); err != nil {
		return err
	}
	if userCount == 0 {
		hashed, err := bcrypt.GenerateFromPassword([]byte("super@xstek2026"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		super := &auth.User{
			ID:           "superuser-001",
			Username:     "admin",
			Email:        "admin@xstek.net",
			PasswordHash: string(hashed),
			Name:         "Super Admin",
			Roles:        []string{"superuser"},
			GrantedPerms: []auth.Permission{},
			DeniedPerms:  []auth.Permission{},
			IsActive:     true,
			CreatedAt:    time.Now(),
		}
		if err := s.UpsertUser(ctx, super); err != nil {
			return fmt.Errorf("seed superuser: %w", err)
		}
		log.Println("pgstore: seeded default superuser")
	}

	if err := s.grantNewPermissionsToSystemRoles(ctx); err != nil {
		return fmt.Errorf("migrate system role permissions: %w", err)
	}
	return nil
}

// grantNewPermissionsToSystemRoles is an idempotent, additive migration:
// every time a new built-in permission is introduced (e.g. the NHS
// integration permissions added alongside internal/nhs), an
// already-seeded database's system roles (roleCount was > 0, so
// auth.DefaultRoles() in EnsureSeed above never ran again) would otherwise
// never receive it. This appends any permission auth.DefaultRoles() says a
// system role should have but the stored row doesn't yet — it only adds,
// never removes, so a superuser's own permission edits on a system role
// (if any) are preserved.
func (s *Store) grantNewPermissionsToSystemRoles(ctx context.Context) error {
	for _, r := range auth.DefaultRoles() {
		_, err := s.pool.Exec(ctx, `
			UPDATE roles SET permissions = (
				SELECT array_agg(DISTINCT p) FROM unnest(permissions || $2::text[]) AS p
			)
			WHERE id = $1 AND is_system = TRUE`,
			r.ID, permsToStrings(r.Permissions))
		if err != nil {
			return fmt.Errorf("role %s: %w", r.ID, err)
		}
	}
	return nil
}

// Close releases the connection pool.
func (s *Store) Close() {
	s.pool.Close()
}

// --- helpers ---

// vecParam renders a []float32 as a pgvector text literal ("[1,2,3]"), or nil
// for a SQL NULL. Always bind through a ::vector cast.
func vecParam(v []float32) any {
	if v == nil {
		return nil
	}
	buf := make([]byte, 0, len(v)*10+2)
	buf = append(buf, '[')
	for i, f := range v {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = strconv.AppendFloat(buf, float64(f), 'f', -1, 32)
	}
	buf = append(buf, ']')
	return string(buf)
}

func permsToStrings(ps []auth.Permission) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = string(p)
	}
	return out
}

func stringsToPerms(ss []string) []auth.Permission {
	out := make([]auth.Permission, len(ss))
	for i, s := range ss {
		out[i] = auth.Permission(s)
	}
	return out
}

// parseSessionTime parses the RFC3339 strings SavedSession carries on the
// wire; zero time on failure so imports never crash on a malformed stamp.
func parseSessionTime(s string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
