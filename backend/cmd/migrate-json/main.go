// Command migrate-json is the one-time import of the legacy JSON file stores
// into Postgres (docs/adr/0001). Idempotent: every record is upserted, so
// re-running is safe. Embeddings are NOT computed here — the server's
// embedding worker backfills them on its next start.
//
// Usage (from backend/):
//
//	go run ./cmd/migrate-json                       # ./data + ./templates/custom
//	go run ./cmd/migrate-json -data cmd/server/data # the other data copy
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"corti-backend/internal/auth"
	"corti-backend/internal/corti"
	"corti-backend/internal/pgstore"
)

func main() {
	dataDir := flag.String("data", "./data", "directory holding users.json / roles.json / sessions.json")
	templatesDir := flag.String("templates", "./templates/custom", "directory holding Corti custom template JSON files")
	databaseURL := flag.String("database", defaultDatabaseURL(), "Postgres connection URL")
	flag.Parse()

	ctx := context.Background()
	store, err := pgstore.Connect(ctx, *databaseURL)
	if err != nil {
		log.Fatalf("connect: %v (is Postgres running? createdb meditrans?)", err)
	}
	defer store.Close()
	if err := store.ApplySchema(ctx); err != nil {
		log.Fatalf("apply schema: %v", err)
	}

	roles := importRoles(ctx, store, filepath.Join(*dataDir, "roles.json"))
	users, userIDs := importUsers(ctx, store, filepath.Join(*dataDir, "users.json"))
	sessions, skipped := importSessions(ctx, store, filepath.Join(*dataDir, "sessions.json"), userIDs)
	templates := importTemplates(store, *templatesDir)

	fmt.Printf("\nImport complete: %d roles, %d users, %d sessions (%d skipped), %d templates.\n",
		roles, users, sessions, skipped, templates)
	fmt.Println("Embeddings are backfilled by the server's embedding worker on next start.")
}

func defaultDatabaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return "postgres://localhost:5432/meditrans?sslmode=disable"
}

func readJSON[T any](path string) ([]T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []T
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return out, nil
}

func importRoles(ctx context.Context, store *pgstore.Store, path string) int {
	roles, err := readJSON[*auth.Role](path)
	if err != nil {
		log.Printf("skipping roles (%v)", err)
		return 0
	}
	count := 0
	for _, r := range roles {
		if err := store.UpsertRole(ctx, r); err != nil {
			log.Printf("role %s: %v", r.ID, err)
			continue
		}
		count++
	}
	return count
}

func importUsers(ctx context.Context, store *pgstore.Store, path string) (int, map[string]bool) {
	ids := map[string]bool{}
	users, err := readJSON[*auth.User](path)
	if err != nil {
		log.Printf("skipping users (%v)", err)
		return 0, ids
	}
	count := 0
	for _, u := range users {
		if err := store.UpsertUser(ctx, u); err != nil {
			log.Printf("user %s: %v", u.Email, err)
			continue
		}
		ids[u.ID] = true
		count++
	}
	return count, ids
}

func importSessions(ctx context.Context, store *pgstore.Store, path string, userIDs map[string]bool) (int, int) {
	sessions, err := readJSON[*auth.SavedSession](path)
	if err != nil {
		log.Printf("skipping sessions (%v)", err)
		return 0, 0
	}
	count, skipped := 0, 0
	for _, s := range sessions {
		if !userIDs[s.UserID] {
			log.Printf("session %s: owner %s not in users.json — skipped", s.ID, s.UserID)
			skipped++
			continue
		}
		if err := store.UpsertSession(ctx, s); err != nil {
			log.Printf("session %s: %v", s.ID, err)
			skipped++
			continue
		}
		count++
	}
	return count, skipped
}

func importTemplates(store *pgstore.Store, dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("skipping templates (%v)", err)
		return 0
	}
	count := 0
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			log.Printf("template %s: %v", entry.Name(), err)
			continue
		}
		var tpl corti.CustomTemplateConfig
		if err := json.Unmarshal(data, &tpl); err != nil {
			log.Printf("template %s: %v", entry.Name(), err)
			continue
		}
		if tpl.Key == "" {
			tpl.Key = entry.Name()[:len(entry.Name())-len(".json")]
		}
		if err := store.SaveTemplate(&tpl); err != nil {
			log.Printf("template %s: %v", tpl.Key, err)
			continue
		}
		count++
	}
	return count
}
