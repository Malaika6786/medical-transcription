# Database Guide

PostgreSQL 17 with the pgvector extension is the system of record for users,
roles, sessions, Corti templates, and every embedding — there is no separate
vector store. Rationale in [adr/0001](adr/0001-postgres-pgvector-replaces-json-storage.md),
[adr/0002](adr/0002-bge-small-384-not-368.md), and [adr/0003](adr/0003-pgx-handwritten-sql-no-orm.md).

Installation steps are in [SETUP.md § 4](SETUP.md#4-step-1--postgresql--pgvector).
This document covers the schema, day-to-day operations, and backup/restore.

---

## 1. Connection

The backend reads `DATABASE_URL` from `backend/.env`:

```env
DATABASE_URL=postgres://localhost:5432/meditrans?sslmode=disable
```

| Deployment | Connection string |
|------------|-------------------|
| Homebrew (macOS) | `postgres://localhost:5432/meditrans?sslmode=disable` — authenticates as your OS user |
| Docker Compose | `postgres://meditrans:meditrans@localhost:5432/meditrans?sslmode=disable` |
| Remote / production | `postgres://USER:PASSWORD@HOST:5432/meditrans?sslmode=require` |

If `DATABASE_URL` is unset the backend falls back to the Homebrew string above.
The helper scripts (`scripts/backup_db.sh`, `scripts/restore_db.sh`) resolve it
the same way: `DATABASE_URL` from the environment, then `backend/.env`, then that
default. They require a URI — key/value DSNs (`host=... dbname=...`) are rejected
with a clear error rather than being silently misparsed.

There is no connection pool tuning; `pgxpool` defaults apply (max 4 connections
or `GOMAXPROCS`, whichever is larger).

---

## 2. Schema

`internal/pgstore/schema.sql` is embedded in the binary and applied at **every**
startup. It is written to be idempotent (`CREATE TABLE IF NOT EXISTS`,
`CREATE INDEX IF NOT EXISTS`), so starting the server against an existing
database is safe and starting against an empty one creates everything.

There is deliberately no migration tool. Once the schema starts changing under
deployed data, adopt goose or golang-migrate — see [adr/0001](adr/0001-postgres-pgvector-replaces-json-storage.md).

### `users`

| Column | Type | Notes |
|--------|------|-------|
| `id` | `TEXT PK` | UUID, or `superuser-001` for the seeded account |
| `email` | `TEXT UNIQUE NOT NULL` | Login identifier |
| `password_hash` | `TEXT NOT NULL` | bcrypt |
| `name` | `TEXT` | |
| `roles` | `TEXT[]` | Role IDs; array rather than a join table — see adr/0001 |
| `granted_permissions` | `TEXT[]` | Per-user grants on top of roles |
| `denied_permissions` | `TEXT[]` | Per-user denials, applied last |
| `is_active` | `BOOLEAN` | Inactive users cannot log in |
| `created_at`, `created_by`, `last_login` | | |

### `roles`

| Column | Type | Notes |
|--------|------|-------|
| `id` | `TEXT PK` | `superuser`, `doctor`, `user` |
| `name`, `description` | `TEXT` | |
| `is_system` | `BOOLEAN` | System roles are seeded and should not be deleted |
| `permissions` | `TEXT[]` | e.g. `ambient.access`, `users.manage` |

### `sessions`

The central table: one row per saved transcription work product.

| Column | Type | Notes |
|--------|------|-------|
| `id` | `TEXT PK` | Client-generated, e.g. `session_1770929147506_kn0l2j02j` |
| `user_id` | `TEXT NOT NULL` | FK → `users(id)` **ON DELETE CASCADE** |
| `type` | `TEXT` | `ambient`, `file-transcription`, or `dictation` |
| `title` | `TEXT` | |
| `transcript` | `TEXT` | Raw transcript text — **PHI** |
| `document` | `JSONB` | Corti-generated clinical document, stored whole |
| `extraction` | `JSONB` | The 15-field AI clinical record, stored whole — **PHI** |
| `interaction_id` | `TEXT` | Corti interaction reference |
| `transcript_embedding` | `vector(384)` | Doc-level vector; stored but deliberately **not** queried by search |
| `extraction_text` | `TEXT` | Exact serialization the extraction vector was computed from, kept so the vector's source is inspectable |
| `extraction_embedding` | `vector(384)` | Queried by search |
| `embedded_at` | `TIMESTAMPTZ` | NULL or older than `updated_at` ⇒ re-embedding pending |
| `created_at`, `updated_at` | `TIMESTAMPTZ` | |

Indexes: `sessions_user_id_idx` (btree), `sessions_extraction_embedding_idx`
(HNSW, `vector_cosine_ops`).

`document` and `extraction` are JSONB rather than normalized tables because the
application only ever reads and writes them whole.

### `transcript_chunks`

| Column | Type | Notes |
|--------|------|-------|
| `id` | `BIGSERIAL PK` | |
| `session_id` | `TEXT NOT NULL` | FK → `sessions(id)` **ON DELETE CASCADE** |
| `chunk_index` | `INT` | Unique together with `session_id` |
| `chunk_text` | `TEXT NOT NULL` | The passage that was embedded — **PHI** |
| `chunk_embedding` | `vector(384) NOT NULL` | |

Indexes: `transcript_chunks_session_idx` (btree),
`transcript_chunks_embedding_idx` (HNSW, `vector_cosine_ops`).

Chunks exist so search can match a specific passage rather than a whole
transcript. They are replaced wholesale each time a session is re-embedded.

### `corti_templates`

| Column | Type | Notes |
|--------|------|-------|
| `key` | `TEXT PK` | e.g. `gp-letter-summary` |
| `name` | `TEXT` | Lifted out of the document for listing |
| `definition` | `JSONB NOT NULL` | Full template configuration |
| `created_at`, `updated_at` | `TIMESTAMPTZ` | |

---

## 3. Seeding on a fresh database

`EnsureSeed` runs after the schema at every startup, but only acts on empty
tables:

- `roles` empty → the three default system roles are inserted.
- `users` empty → a superuser is created: **`admin@xstek.net` /
  `super@xstek2026`**, ID `superuser-001`.

Neither ever overwrites existing rows. Change that password on any machine that
is not a local laptop.

---

## 4. Embedding lifecycle

Session saves never wait on the embedding model — a model outage cannot fail a
save.

```
POST /api/sessions
   └─► row written, embedded_at left NULL
   └─► embedding.Worker.Notify()          (non-blocking)
            │
            ▼
   worker picks up sessions WHERE embedded_at IS NULL OR embedded_at < updated_at
   (batches of 8, on notify + every 60s + once at startup)
            │
            ▼
   one /v1/embeddings call per session: all transcript chunks
   + the whole transcript + the extraction serialization
            │
            ▼
   StoreSessionEmbeddings() — single transaction:
     UPDATE sessions SET transcript_embedding, extraction_text,
                         extraction_embedding, embedded_at = now()
     DELETE FROM transcript_chunks WHERE session_id = ...
     INSERT the new chunks
```

The startup pass doubles as the backfill path for data imported by
`cmd/migrate-json`, and as the recovery path after any embedder outage. On error
the worker stops the current pass and waits for the next tick instead of
hot-looping against a down endpoint.

Check for pending work:

```bash
psql -d meditrans -Atc \
  "SELECT count(*) FROM sessions WHERE embedded_at IS NULL OR embedded_at < updated_at;"
```

### Forcing a re-embed

Clearing `embedded_at` makes the worker pick the row up again within 60 seconds:

```sql
UPDATE sessions SET embedded_at = NULL WHERE id = 'session_...';  -- one session
UPDATE sessions SET embedded_at = NULL;                            -- everything
```

Required after any change to the embedding model, the chunking logic, or the
extraction serialization.

---

## 5. How semantic search queries the data

`POST /api/search` embeds the query (with the bge retrieval prefix), then runs
one SQL statement: transcript chunks `UNION ALL` extraction embeddings, filtered
to the caller's own sessions, `DISTINCT ON (session_id)` keeping the best hit per
session, ordered by cosine similarity `1 - (embedding <=> query)`, with a
minimum score of 0.35 and a default limit of 10 (max 25).

The doc-level `transcript_embedding` is intentionally excluded: it is redundant
with its own chunks and would double-count sessions.

---

## 6. Data retention behaviour worth knowing

**Saving a session when a user already has 5 deletes their oldest one.**
`SaveSession` counts the user's rows and, at the cap
(`auth.MaxSessionsPerUser = 5`), deletes the oldest by `updated_at` inside the
same transaction. The `ON DELETE CASCADE` on `transcript_chunks` removes its
chunks too. This is enforced in the backend, not just the UI, and there is no
soft delete or audit trail — **an evicted session is gone unless it is in a
backup**. Deleting a user cascades to all of their sessions the same way.

---

## 7. Backup

```bash
bash scripts/backup_db.sh
```

Writes three files into `backups/` (mode 700, gitignored):

| File | Contents |
|------|----------|
| `meditrans-YYYYmmdd-HHMMSS.dump` | `pg_dump --format=custom` — compressed, selectively restorable |
| `meditrans-YYYYmmdd-HHMMSS.manifest.txt` | Row counts per table, server and pg_dump versions, pgvector version, UTC timestamp |
| `meditrans-YYYYmmdd-HHMMSS.sql` | Only with `--plain`: readable SQL, useful for diffing or partial recovery |

Options:

| Flag | Effect |
|------|--------|
| `--plain` | Also write the plain-SQL dump |
| `--verify` | Restore the dump into a scratch database, compare row counts against the manifest, drop the scratch database |
| `--out DIR` | Write somewhere other than `backups/` |

Dumps are taken with `--no-owner --no-privileges` so they restore under any
local role — the Homebrew setup owns objects as your OS user, Docker owns them
as `meditrans`, and a dump should move between the two.

A verified backup looks like this:

```
Database : meditrans (server 17.10, pg_dump 17.10)
Output   : backups/meditrans-20260812-215036.dump
corti_templates|2
roles|3
sessions|6
transcript_chunks|6
users|4
pgvector=0.8.6
Verifying restore into scratch database meditrans_restorecheck_1786542637 ...
Restore verified: row counts match the source database.
```

### What is and is not in a backup

Included: all five tables with data, indexes, constraints, and the
`CREATE EXTENSION vector` statement. Embeddings are dumped as plain vector
literals, so no re-embedding is needed after a restore.

Not included: `backend/.env` (secrets), `backend/data/*.json` (legacy import
source), `backend/templates/custom/` (Corti templates already imported into the
DB), and the Ollama models. Back those up separately — the `.env` in particular
is unrecoverable and gitignored.

### Security

Dumps contain patient transcripts, AI extractions, and bcrypt password hashes.
The script writes them mode 600 in a mode 700 directory, and `backups/` plus
`*.dump` are in `.gitignore`. Do not commit them, attach them to tickets, or
sync them to shared cloud folders.

### Scheduling

macOS (`launchd`), daily at 02:00 — write to
`~/Library/LaunchAgents/com.xstek.meditrans-backup.plist` and load it with
`launchctl load ~/Library/LaunchAgents/com.xstek.meditrans-backup.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.xstek.meditrans-backup</string>
  <key>ProgramArguments</key>
  <array>
    <string>/bin/bash</string>
    <string>/ABSOLUTE/PATH/TO/medical-transcription/scripts/backup_db.sh</string>
  </array>
  <key>StartCalendarInterval</key>
  <dict><key>Hour</key><integer>2</integer><key>Minute</key><integer>0</integer></dict>
  <key>StandardOutPath</key><string>/tmp/meditrans-backup.log</string>
  <key>StandardErrorPath</key><string>/tmp/meditrans-backup.log</string>
</dict>
</plist>
```

Linux (`cron`):

```cron
0 2 * * * cd /path/to/medical-transcription && bash scripts/backup_db.sh >> /var/log/meditrans-backup.log 2>&1
```

Retention — keep 14 days of dumps:

```bash
find backups -name '*.dump' -mtime +14 -delete
find backups -name '*.manifest.txt' -mtime +14 -delete
```

Run `--verify` at least on the first backup of a new machine and after any
PostgreSQL upgrade. A backup that has never been restored is a hypothesis.

---

## 8. Restore

```bash
bash scripts/restore_db.sh backups/meditrans-20260812-215036.dump
```

This **drops and recreates** the target database, so it prompts for confirmation
(`--force` skips it). It terminates existing connections first — stop the backend
anyway, or it will reconnect mid-restore. Afterwards it prints the restored row
counts next to the manifest's expected counts.

| Flag | Effect |
|------|--------|
| `--into NAME` | Restore into a different database (safe way to inspect a backup) |
| `--force` | Skip the confirmation prompt |

Inspect a backup without touching the live database:

```bash
bash scripts/restore_db.sh --into meditrans_check backups/meditrans-....dump
psql -d meditrans_check -c 'SELECT id, title, updated_at FROM sessions ORDER BY updated_at DESC;'
dropdb meditrans_check
```

Recover a single table from a custom-format dump:

```bash
pg_restore --data-only --table=sessions --dbname="$DATABASE_URL" backups/meditrans-....dump
```

Manual equivalent of the script:

```bash
dropdb meditrans && createdb meditrans
pg_restore --dbname=meditrans --no-owner --no-privileges backups/meditrans-....dump
```

After a restore, start the backend: it re-applies the schema (a no-op) and the
embedding worker backfills anything with a NULL `embedded_at`.

**Moving to another machine**: restoring is enough. The target only needs
PostgreSQL 17 with pgvector installed — the dump carries the
`CREATE EXTENSION vector` statement, and the vectors come across as data.

---

## 9. Routine operations

```bash
# Sizes
psql -d meditrans -Atc "SELECT pg_size_pretty(pg_database_size('meditrans'));"
psql -d meditrans -c "SELECT relname, pg_size_pretty(pg_total_relation_size(relid))
                      FROM pg_stat_user_tables ORDER BY pg_total_relation_size(relid) DESC;"

# Sessions per user
psql -d meditrans -c "SELECT u.email, count(s.id)
                      FROM users u LEFT JOIN sessions s ON s.user_id = u.id
                      GROUP BY u.email ORDER BY 2 DESC;"

# Embedding coverage
psql -d meditrans -c "SELECT count(*) FILTER (WHERE embedded_at IS NOT NULL) AS embedded,
                             count(*) FILTER (WHERE embedded_at IS NULL)     AS pending,
                             count(*) AS total
                      FROM sessions;"

# Reset a forgotten password (bcrypt hash must be generated by the app —
# simplest path is deleting the user and letting the superuser recreate them)
psql -d meditrans -c "SELECT id, email, roles, is_active FROM users;"
```

pgAdmin 4 is the GUI option; it is a human-facing client only, never part of the
running application. Connect it to `127.0.0.1:5432`, database `meditrans`.

### Legacy JSON import

```bash
cd backend
go run ./cmd/migrate-json                        # ./data + ./templates/custom
go run ./cmd/migrate-json -data cmd/server/data  # the other data copy
```

Idempotent (everything is upserted), computes no embeddings — the worker
backfills them on the next backend start. Sessions whose `user_id` does not
resolve to an imported user are skipped and reported.

---

## 10. Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `Failed to connect to Postgres` at startup | Server down, or database missing | `brew services start postgresql@17 && createdb meditrans` |
| `type "vector" does not exist` | pgvector not installed for this PG major version | `brew install pgvector` / `apt install postgresql-17-pgvector`, restart PostgreSQL |
| `expected 384 dimensions, got N` | `EMBED_MODEL` is not bge-small-en-v1.5 | Restore the correct model; changing dimension means re-embedding everything and altering every `vector(384)` column |
| Search returns nothing on a populated database | Embeddings pending or embedder down | Check the pending count (§ 4) and the backend log |
| `pg_dump: server version mismatch` | Client older than the server | Use the matching client: `brew link --force postgresql@17` |
| `database "meditrans" is being accessed by other users` during restore | Backend still connected | Stop the backend; the script also terminates connections |
| Sessions disappear on their own | 5-session cap evicted the oldest (§ 6) | Restore from a backup; the cap is `auth.MaxSessionsPerUser` |
| `DATABASE_URL must be a URI...` from a script | Key/value DSN supplied | Use `postgres://user:pass@host:port/dbname?opts` |
