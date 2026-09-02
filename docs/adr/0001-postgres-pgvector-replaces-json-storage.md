# Postgres + pgvector replaces JSON file storage

Users, roles, sessions, and Corti templates were persisted as JSON files in `cmd/server/data/` — fine for a demo, but the program now needs permanent storage plus semantic search. A single PostgreSQL database is the system of record for all of it, including every embedding — no separate vector store. Deployment: the `pgvector/pgvector` Docker image (`docker-compose.yml`) where Docker exists; the current dev machine has no Docker, so it runs Homebrew `postgresql@17` + `pgvector` instead. pgAdmin is the GUI either way. The JSON files survive only as the source for a one-time `cmd/migrate-json` import; dual-mode JSON/DB storage was considered and rejected as complexity with no payoff.

## Consequences

- Extraction and Document are stored as JSONB, not normalized tables: the app only ever reads/writes them whole, never queries individual fields. Same reasoning for roles/permissions living as arrays on `users` instead of join tables.
- The schema is an idempotent `schema.sql` embedded in the binary and applied at startup. No migration tool (goose/golang-migrate) until the schema starts evolving under deployed data.
