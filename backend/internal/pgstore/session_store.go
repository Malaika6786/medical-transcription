package pgstore

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"

	"corti-backend/internal/ai"
	"corti-backend/internal/auth"
)

const sessionColumns = `id, user_id, type, title, transcript, document, extraction, interaction_id, created_at, updated_at`

func scanSession(row pgx.Row) (*auth.SavedSession, error) {
	var (
		sess                 auth.SavedSession
		document, extraction []byte
		createdAt, updatedAt time.Time
	)
	err := row.Scan(&sess.ID, &sess.UserID, &sess.Type, &sess.Title, &sess.Transcript, &document, &extraction, &sess.InteractionID, &createdAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, auth.ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(document) > 0 {
		var doc auth.SavedDocument
		if err := json.Unmarshal(document, &doc); err == nil {
			sess.Document = &doc
		}
	}
	if len(extraction) > 0 {
		var ext ai.ExtractionResult
		if err := json.Unmarshal(extraction, &ext); err == nil {
			sess.Extraction = &ext
		}
	}
	sess.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	sess.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return &sess, nil
}

// jsonbParam marshals v to JSONB bytes, or nil for SQL NULL when v is a nil pointer.
func jsonbParam(v any) (any, error) {
	switch t := v.(type) {
	case *auth.SavedDocument:
		if t == nil {
			return nil, nil
		}
	case *ai.ExtractionResult:
		if t == nil {
			return nil, nil
		}
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// GetSessionsByUserID returns all sessions for a user, newest first.
func (s *Store) GetSessionsByUserID(userID string) []*auth.SavedSession {
	rows, err := s.pool.Query(context.Background(),
		`SELECT `+sessionColumns+` FROM sessions WHERE user_id = $1 ORDER BY updated_at DESC`, userID)
	if err != nil {
		log.Printf("pgstore: list sessions: %v", err)
		return []*auth.SavedSession{}
	}
	defer rows.Close()
	sessions := []*auth.SavedSession{}
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			log.Printf("pgstore: scan session: %v", err)
			continue
		}
		sessions = append(sessions, sess)
	}
	return sessions
}

// GetSessionByID returns a specific session owned by userID.
func (s *Store) GetSessionByID(sessionID, userID string) (*auth.SavedSession, error) {
	row := s.pool.QueryRow(context.Background(),
		`SELECT `+sessionColumns+` FROM sessions WHERE id = $1 AND user_id = $2`, sessionID, userID)
	return scanSession(row)
}

// SaveSession creates or updates a session, preserving the JSON store's
// semantics: a nil Extraction never overwrites a stored one. There is no
// longer a per-user session cap (previously auth.MaxSessionsPerUser, which
// evicted the oldest session once a user hit 5) — removed at the user's
// request; save as many sessions as you want.
func (s *Store) SaveSession(session *auth.SavedSession) (*auth.SavedSession, error) {
	ctx := context.Background()
	docParam, err := jsonbParam(session.Document)
	if err != nil {
		return nil, err
	}
	extParam, err := jsonbParam(session.Extraction)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	now := time.Now().UTC()

	tag, err := tx.Exec(ctx, `
		UPDATE sessions SET
			title = $3,
			transcript = $4,
			document = $5,
			extraction = COALESCE($6, extraction),
			updated_at = $7
		WHERE id = $1 AND user_id = $2`,
		session.ID, session.UserID, session.Title, session.Transcript, docParam, extParam, now)
	if err != nil {
		return nil, err
	}

	if tag.RowsAffected() == 0 {
		// New session — no cap, no eviction.
		if _, err := tx.Exec(ctx, `
			INSERT INTO sessions (id, user_id, type, title, transcript, document, extraction, interaction_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)`,
			session.ID, session.UserID, session.Type, session.Title, session.Transcript,
			docParam, extParam, session.InteractionID, now); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetSessionByID(session.ID, session.UserID)
}

// UpsertSession inserts or replaces a full session record (used by cmd/migrate-json).
// No cap eviction — imports are trusted as-is.
func (s *Store) UpsertSession(ctx context.Context, session *auth.SavedSession) error {
	docParam, err := jsonbParam(session.Document)
	if err != nil {
		return err
	}
	extParam, err := jsonbParam(session.Extraction)
	if err != nil {
		return err
	}
	createdAt := parseSessionTime(session.CreatedAt)
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := parseSessionTime(session.UpdatedAt)
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO sessions (id, user_id, type, title, transcript, document, extraction, interaction_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			type = EXCLUDED.type,
			title = EXCLUDED.title,
			transcript = EXCLUDED.transcript,
			document = EXCLUDED.document,
			extraction = EXCLUDED.extraction,
			interaction_id = EXCLUDED.interaction_id,
			updated_at = EXCLUDED.updated_at`,
		session.ID, session.UserID, session.Type, session.Title, session.Transcript,
		docParam, extParam, session.InteractionID, createdAt, updatedAt)
	return err
}

// UpdateSessionDocument updates only the document of a session.
func (s *Store) UpdateSessionDocument(sessionID, userID string, document *auth.SavedDocument) error {
	docParam, err := jsonbParam(document)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(context.Background(),
		`UPDATE sessions SET document = $3, updated_at = $4 WHERE id = $1 AND user_id = $2`,
		sessionID, userID, docParam, time.Now().UTC())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return auth.ErrSessionNotFound
	}
	return nil
}

// DeleteSession removes a session (chunks cascade).
func (s *Store) DeleteSession(sessionID, userID string) error {
	tag, err := s.pool.Exec(context.Background(),
		`DELETE FROM sessions WHERE id = $1 AND user_id = $2`, sessionID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return auth.ErrSessionNotFound
	}
	return nil
}
