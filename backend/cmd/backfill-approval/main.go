// Command backfill-approval is a one-time, manually-run migration for the
// signup-approval workflow: it retroactively marks every existing account
// (except the hardcoded superuser) as "pending", requiring a superuser to
// approve it before it works again. Run this exactly once, right after
// deploying the schema/code that adds the approval workflow — NOT on every
// server startup, unlike the idempotent DDL in internal/pgstore/schema.sql.
//
// Each account's actual `roles` are left untouched (only `status` and
// `requested_role` change) — approving is just a status flip, not a
// re-grant, so nothing here is destructive or hard to undo.
//
// Usage (from backend/):
//
//	go run ./cmd/backfill-approval
//	go run ./cmd/backfill-approval -database postgres://...
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"corti-backend/internal/pgstore"
)

func main() {
	databaseURL := flag.String("database", defaultDatabaseURL(), "Postgres connection URL")
	flag.Parse()

	ctx := context.Background()
	store, err := pgstore.Connect(ctx, *databaseURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer store.Close()

	affected, err := store.BackfillPendingApproval(ctx)
	if err != nil {
		log.Fatalf("backfill: %v", err)
	}

	fmt.Printf("Marked %d existing account(s) pending. The superuser (admin@xstek.net) was left untouched.\n", affected)
	fmt.Println("Each account's real roles are preserved — approving via POST /api/users/:id/approve just flips status back, nothing is re-granted from scratch.")
}

func defaultDatabaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return "postgres://localhost:5432/meditrans?sslmode=disable"
}
