# Optimization Review

Findings from a read-only review of the application as of 2026-08-12. **No code
was changed.** Each item states what was measured, why it matters, and what the
fix would be, so the work can be scheduled deliberately.

Measurements were taken on the development machine (macOS, Apple Silicon,
PostgreSQL 17.10 + pgvector 0.8.6, 6 sessions / 6 transcript chunks / 4 users).
At this data volume nothing is slow today — the ranking below is about what
degrades first as the dataset grows.

---

## Summary

| # | Finding | Area | Impact | Effort |
|---|---------|------|--------|--------|
| 1 | Session re-embeds in full on every save, even a title edit | Backend | High (cost + latency) | Low |
| 2 | HNSW indexes are maintained but never used by the search query | Database | Medium (write cost now, read cost later) | Medium |
| 3 | `GET /api/sessions` returns full transcripts and extractions for the list view | API | Medium | Low |
| 4 | No HTTP response compression | API | Medium | Low |
| 5 | One failing session stalls the whole embedding pass for 60s | Backend | Medium | Low |
| 6 | Embedding worker issues one HTTP call per session | Backend | Low | Low |
| 7 | Connection pool left at library defaults | Database | Low | Low |
| 8 | Legacy JSON data duplicated in two tracked directories | Hygiene | Low (but see the note) | Low |
| 9 | Compiled `vite.config.js` / `.d.ts` committed next to the source | Hygiene | Low | Low |
| 10 | `npm run lint` cannot run — eslint missing from devDependencies | Tooling | Low | Low |
| 11 | `LOG_LEVEL=debug` is the shipped default | Ops | Low | Trivial |
| 12 | Controller script: unchecked `rg` dependency, `gcloud` required to stop | Ops | Low | Low |

Already in good shape, for the record: the frontend bundle is properly code-split
(pdfmake, its fonts, and docx are all dynamic imports), Corti OAuth tokens are
cached with a refresh buffer, embeddings are computed off the request path, and
the search SQL is a single round trip rather than an N+1.

---

## 1. Sessions re-embed in full on every save

**Where** `internal/embedding/worker.go`, `internal/pgstore/search.go`

The worker selects sessions `WHERE embedded_at IS NULL OR embedded_at < updated_at`.
Any write bumps `updated_at`, so renaming a session, attaching a document, or
re-saving unchanged text queues a complete re-embedding: every chunk is deleted
and reinserted, and the whole transcript plus the extraction are re-sent to the
model.

On the GPU VM that is a few hundred milliseconds. On a laptop with local Ollama
it is seconds of CPU per save, and against a metered hosted embedding API it is
money spent to recompute identical vectors.

**Fix** Store a content hash (transcript + extraction serialization) alongside
`embedded_at`, and skip the session when the hash is unchanged. One extra
column, one comparison in `PendingEmbeddingSessions`.

## 2. HNSW indexes are maintained but never used

**Where** `internal/pgstore/schema.sql`, `SearchSessions`

`sessions_extraction_embedding_idx` and `transcript_chunks_embedding_idx` are
HNSW indexes on `vector_cosine_ops`. `EXPLAIN (ANALYZE, BUFFERS)` on the live
search query shows neither is touched:

```
->  Index Scan using sessions_user_id_idx on sessions sess_1
->  Bitmap Heap Scan on transcript_chunks c
      Recheck Cond: (session_id = sess_1.id)
```

This is a direct consequence of the query shape, and the shape is deliberate:
results are scoped to `sess.user_id = $2`, and the distance ordering happens
after the `UNION ALL` and `DISTINCT ON`. An approximate-nearest-neighbour index
can only serve a plain `ORDER BY embedding <=> query LIMIT k`, so PostgreSQL
falls back to an exact scan of that user's chunks — correct results, no recall
loss, but linear in the user's data.

Today: 6 chunks, 0.5 ms. The indexes still cost write time and storage on every
re-embed and return nothing.

**Options**

- *Leave as is* and revisit at ~10k chunks per user. Exact search over a few
  thousand 384-dim vectors is milliseconds.
- *Drop the two HNSW indexes* to stop paying for maintenance until they are read.
- *Restructure* to a two-stage query: an indexed ANN candidate fetch (`ORDER BY
  <=> LIMIT k*5`, no user filter), then filter by user and re-rank. Faster at
  scale, but recall depends on the over-fetch factor.

Recommendation: leave the schema alone, record the measurement, and re-run the
EXPLAIN once a user exceeds a few thousand chunks. Do not remove the indexes
without agreeing which of the three paths is intended.

## 3. `GET /api/sessions` returns everything

**Where** `handlers/sessions_handler.go:31`, `pgstore/session_store.go:16`

`HandleListSessions` returns whole session rows — transcript, `document` JSONB,
and `extraction` JSONB — for the sessions list. With the 5-session cap the
payload stays bounded, but it is still hundreds of KB of clinical text sent to
render a list of titles and dates, and it puts PHI on the wire for a view that
does not display it.

**Fix** A list projection (`id, type, title, interaction_id, created_at,
updated_at`, plus booleans for "has document"/"has extraction"). The detail
endpoint already exists for the full record.

## 4. No response compression

**Where** `cmd/server/main.go` middleware stack

`middleware.RecoverConfig`, `RequestLogger`, and `CORSConfig` are registered;
Fiber's `compress` middleware is not. Transcript and extraction JSON compresses
roughly 5–10x. One line, and it compounds with finding 3.

## 5. One bad session stalls the embedding pass

**Where** `internal/embedding/worker.go`

```go
if err := w.embedSession(ctx, p); err != nil {
    log.Printf(...)
    return          // abandons the rest of the batch
}
```

Backing off instead of hot-looping against a down endpoint is the right instinct,
but the same `return` fires for a session-specific failure — one oversized or
malformed transcript blocks every other pending session until the next 60s tick,
then blocks them again, indefinitely.

**Fix** Distinguish transport errors (back off, as now) from per-session errors
(log, count, continue). A `failed_at` / attempt counter would also stop a
permanently broken row from being retried forever.

## 6. One HTTP call per session

**Where** `worker.processPending` iterates a batch of 8 sessions, calling
`EmbedPassages` once per session. The endpoint accepts arrays, so a batch could
be one call. Marginal against local Ollama, meaningful against a hosted API
where per-request overhead dominates. Only worth doing alongside finding 1.

## 7. Connection pool defaults

**Where** `pgstore.Connect` uses `pgxpool.New` with no configuration, so the pool
caps at `max(4, GOMAXPROCS)` connections. With WebSocket sessions, the embedding
worker, and search all competing, an explicit `MaxConns` / `MinConns` /
`MaxConnIdleTime` would make behaviour predictable under load rather than
machine-dependent.

## 8. Legacy JSON data duplicated in two tracked directories

**Where** `backend/data/` and `backend/cmd/server/data/`

Both hold `users.json`, `roles.json`, `sessions.json`; both are committed. The
sessions files contain transcript text and the users files contain bcrypt
password hashes. Since [adr/0001](adr/0001-postgres-pgvector-replaces-json-storage.md)
these are read only by `cmd/migrate-json` — nothing writes them anymore, so this
is historical rather than ongoing exposure, and the transcripts present look like
demo dictation. Still worth resolving deliberately: confirm the content is
synthetic, keep one copy as the import fixture, and delete the other.

Note that `git rm` alone does not remove the content from history.

## 9. Compiled config artifacts are committed

`frontend/vite.config.js` and `frontend/vite.config.d.ts` are tracked alongside
`vite.config.ts`; `vitest.config.js` / `.d.ts` and two `.tsbuildinfo` files exist
untracked. Vite reads the `.ts` source, so the compiled copies are stale
duplicates waiting to confuse someone. Delete them and add
`*.tsbuildinfo`, `vite.config.js`, `vite.config.d.ts` to `frontend/.gitignore`.

## 10. `npm run lint` has never worked

`package.json` defines a lint script, but eslint is not in `devDependencies`, so
the command fails on a clean install. Either add eslint with a config or remove
the script — leaving it there means every new contributor hits a failure that
looks like their fault. Type checking still runs through `npm run build`
(`vue-tsc -b`).

## 11. `LOG_LEVEL=debug` ships as the default

Both `.env.example` and the code default to `debug`. Combined with the request
logger, production logs will be noisy and may contain more request detail than
intended. Default to `info` in deployed environments.

## 12. Controller script portability

**Where** `~/.codex/skills/manage-medical-transcription/scripts/project_ctl.sh`

- `homebrew_service_running` pipes into `rg`, which is never checked by
  `require_command`. Without ripgrep the check silently reports a running
  PostgreSQL as stopped.
- `stop_project` calls `require_command gcloud` unconditionally, so on a machine
  with no GCP setup — the normal case when the AI endpoint is hosted —
  `stop` fails before it stops anything.

Both are two-line fixes in the script. Documented in
[SETUP.md § 11](SETUP.md#11-optional--one-command-startstop) so they do not
surprise anyone setting up a second machine.

---

## Measurement commands

Re-run these before and after any change so the effect is visible rather than
assumed.

```bash
# Does the search query use the HNSW indexes yet?
psql -d meditrans -c "EXPLAIN (ANALYZE, BUFFERS) <the SearchSessions query>"

# Table and index sizes
psql -d meditrans -c "SELECT relname, pg_size_pretty(pg_total_relation_size(relid))
                      FROM pg_stat_user_tables ORDER BY pg_total_relation_size(relid) DESC;"
psql -d meditrans -c "SELECT indexrelname, idx_scan, pg_size_pretty(pg_relation_size(indexrelid))
                      FROM pg_stat_user_indexes ORDER BY pg_relation_size(indexrelid) DESC;"

# Embedding backlog
psql -d meditrans -Atc "SELECT count(*) FROM sessions
                        WHERE embedded_at IS NULL OR embedded_at < updated_at;"

# Frontend bundle composition
cd frontend && npm run build && ls -lhS dist/assets | head -20

# API payload size for the sessions list
curl -s -o /dev/null -w '%{size_download} bytes\n' \
  http://localhost:8090/api/sessions -H "Authorization: Bearer $JWT"
```

`idx_scan` in the second query is the quickest way to confirm finding 2 on any
machine: it stays at 0 for both HNSW indexes.

---

## Suggested order

1. Findings 1 and 5 — embedding worker correctness and cost. Small, self-contained.
2. Findings 3 and 4 — list projection plus compression. Immediately visible in
   payload sizes, and reduces PHI on the wire.
3. Findings 9, 10, 11 — repository and configuration hygiene.
4. Finding 8 — decide and act on the duplicated data directories.
5. Finding 2 — revisit only when the EXPLAIN or `idx_scan` numbers justify it.
6. Findings 6, 7, 12 — opportunistic.
