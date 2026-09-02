package pgstore

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/jackc/pgx/v5"

	"corti-backend/internal/corti"
)

// GetTemplate returns a Corti custom template by key.
func (s *Store) GetTemplate(key string) (*corti.CustomTemplateConfig, error) {
	var definition []byte
	err := s.pool.QueryRow(context.Background(),
		`SELECT definition FROM corti_templates WHERE key = $1`, key).Scan(&definition)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTemplateNotFound
	}
	if err != nil {
		return nil, err
	}
	var tpl corti.CustomTemplateConfig
	if err := json.Unmarshal(definition, &tpl); err != nil {
		return nil, err
	}
	return &tpl, nil
}

// ListTemplates returns all Corti custom templates.
func (s *Store) ListTemplates() ([]*corti.CustomTemplateConfig, error) {
	rows, err := s.pool.Query(context.Background(),
		`SELECT definition FROM corti_templates ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	templates := []*corti.CustomTemplateConfig{}
	for rows.Next() {
		var definition []byte
		if err := rows.Scan(&definition); err != nil {
			return nil, err
		}
		var tpl corti.CustomTemplateConfig
		if err := json.Unmarshal(definition, &tpl); err != nil {
			log.Printf("pgstore: skipping unparseable template: %v", err)
			continue
		}
		templates = append(templates, &tpl)
	}
	return templates, nil
}

// TemplateExists reports whether a template key is taken.
func (s *Store) TemplateExists(key string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM corti_templates WHERE key = $1)`, key).Scan(&exists)
	return exists, err
}

// SaveTemplate inserts or updates a template, storing the full config as JSONB.
func (s *Store) SaveTemplate(tpl *corti.CustomTemplateConfig) error {
	definition, err := json.Marshal(tpl)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(context.Background(), `
		INSERT INTO corti_templates (key, name, definition, created_at, updated_at)
		VALUES ($1, $2, $3, now(), now())
		ON CONFLICT (key) DO UPDATE SET
			name = EXCLUDED.name,
			definition = EXCLUDED.definition,
			updated_at = now()`,
		tpl.Key, tpl.Name, definition)
	return err
}

// DeleteTemplate removes a template by key.
func (s *Store) DeleteTemplate(key string) error {
	tag, err := s.pool.Exec(context.Background(),
		`DELETE FROM corti_templates WHERE key = $1`, key)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}
