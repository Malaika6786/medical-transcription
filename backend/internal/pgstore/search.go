package pgstore

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	"corti-backend/internal/ai"
)

// SearchResult is one ranked hit of a semantic search: the best-matching
// passage (transcript chunk or extraction serialization) per session.
type SearchResult struct {
	SessionID string  `json:"sessionId"`
	Title     string  `json:"title"`
	Type      string  `json:"type"`
	UpdatedAt string  `json:"updatedAt"`
	Source    string  `json:"source"` // "transcript" or "extraction"
	Snippet   string  `json:"snippet"`
	Score     float64 `json:"score"` // cosine similarity in [0,1]-ish
}

// SearchSessions runs a cosine-similarity search over the caller's own
// sessions, querying chunk embeddings ∪ extraction embeddings and keeping the
// best hit per session (docs/adr/0002 explains why the doc-level
// transcript_embedding is deliberately not queried).
func (s *Store) SearchSessions(ctx context.Context, userID string, queryVec []float32, limit int, minScore float64) ([]SearchResult, error) {
	rows, err := s.pool.Query(ctx, `
		WITH hits AS (
			SELECT c.session_id,
			       'transcript'::text AS source,
			       c.chunk_text AS snippet,
			       1 - (c.chunk_embedding <=> $1::vector) AS score
			FROM transcript_chunks c
			JOIN sessions sess ON sess.id = c.session_id
			WHERE sess.user_id = $2
			UNION ALL
			SELECT sess.id,
			       'extraction',
			       COALESCE(sess.extraction_text, ''),
			       1 - (sess.extraction_embedding <=> $1::vector)
			FROM sessions sess
			WHERE sess.user_id = $2 AND sess.extraction_embedding IS NOT NULL
		),
		best AS (
			SELECT DISTINCT ON (session_id) session_id, source, snippet, score
			FROM hits
			ORDER BY session_id, score DESC
		)
		SELECT b.session_id, sess.title, sess.type, sess.updated_at, b.source, b.snippet, b.score
		FROM best b
		JOIN sessions sess ON sess.id = b.session_id
		WHERE b.score >= $4
		ORDER BY b.score DESC
		LIMIT $3`,
		vecParam(queryVec), userID, limit, minScore)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []SearchResult{}
	for rows.Next() {
		var (
			r         SearchResult
			updatedAt time.Time
		)
		if err := rows.Scan(&r.SessionID, &r.Title, &r.Type, &updatedAt, &r.Source, &r.Snippet, &r.Score); err != nil {
			return nil, err
		}
		r.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		results = append(results, r)
	}
	return results, rows.Err()
}

// PendingSession is a session whose embeddings are missing or stale
// (embedded_at is NULL or older than updated_at).
type PendingSession struct {
	ID         string
	Transcript string
	Extraction *ai.ExtractionResult
}

// PendingEmbeddingSessions returns up to limit sessions needing (re-)embedding,
// oldest first. Also the backfill path for imported JSON data.
func (s *Store) PendingEmbeddingSessions(ctx context.Context, limit int) ([]PendingSession, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, transcript, extraction
		FROM sessions
		WHERE (embedded_at IS NULL OR embedded_at < updated_at)
		  AND (transcript <> '' OR extraction IS NOT NULL)
		ORDER BY updated_at ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pending := []PendingSession{}
	for rows.Next() {
		var (
			p          PendingSession
			extraction []byte
		)
		if err := rows.Scan(&p.ID, &p.Transcript, &extraction); err != nil {
			return nil, err
		}
		if len(extraction) > 0 {
			var ext ai.ExtractionResult
			if err := json.Unmarshal(extraction, &ext); err == nil {
				p.Extraction = &ext
			}
		}
		pending = append(pending, p)
	}
	return pending, rows.Err()
}

// ChunkEmbedding is one transcript chunk with its vector, ready to persist.
type ChunkEmbedding struct {
	Index  int
	Text   string
	Vector []float32
}

// StoreSessionEmbeddings atomically replaces a session's embeddings: the
// doc-level transcript vector, the extraction text + vector, and all chunks.
// Nil vectors store as NULL. Marks the session embedded as of now.
func (s *Store) StoreSessionEmbeddings(ctx context.Context, sessionID string, transcriptVec []float32, extractionText string, extractionVec []float32, chunks []ChunkEmbedding) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `
		UPDATE sessions SET
			transcript_embedding = $2::vector,
			extraction_text = NULLIF($3, ''),
			extraction_embedding = $4::vector,
			embedded_at = now()
		WHERE id = $1`,
		sessionID, vecParam(transcriptVec), extractionText, vecParam(extractionVec)); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM transcript_chunks WHERE session_id = $1`, sessionID); err != nil {
		return err
	}
	if len(chunks) > 0 {
		batch := &pgx.Batch{}
		for _, c := range chunks {
			batch.Queue(`
				INSERT INTO transcript_chunks (session_id, chunk_index, chunk_text, chunk_embedding)
				VALUES ($1, $2, $3, $4::vector)`,
				sessionID, c.Index, c.Text, vecParam(c.Vector))
		}
		if err := tx.SendBatch(ctx, batch).Close(); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
