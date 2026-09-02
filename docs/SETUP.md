# Setup Guide

Everything needed to bring this project up from a clean machine: PostgreSQL +
pgvector, the AI models, the Go backend, and the Vue frontend. Follow the steps
in order — later steps assume the earlier ones passed their verification check.

Companion documents:

| Document | Covers |
|----------|--------|
| [DATABASE.md](DATABASE.md) | Schema reference, backup and restore, database operations |
| [AI-MODELS.md](AI-MODELS.md) | Every AI model and service the app depends on, and how each is provisioned |
| [OPTIMIZATION.md](OPTIMIZATION.md) | Performance review and prioritized improvement list |
| [adr/](adr/) | Why the storage, embedding, and SQL decisions were made |

---

## 1. What you are setting up

```
                    ┌──────────────────────────────────────────┐
  Browser ──────────►  Vue 3 + Vuetify frontend  (Vite :5173)  │
                    └───────────────┬──────────────────────────┘
                                    │  /api/* proxied by Vite
                                    ▼
                    ┌──────────────────────────────────────────┐
                    │  Go + Fiber backend        (:8090)       │
                    └───┬───────────────┬──────────────────┬───┘
                        │               │                  │
          ┌─────────────▼──┐   ┌────────▼─────────┐   ┌────▼───────────────┐
          │ PostgreSQL 17  │   │ Chat LLM         │   │ Corti API (cloud)  │
          │ + pgvector     │   │ (OpenAI-compat)  │   │ speech-to-text     │
          │ :5432          │   │ :11434 or :8443  │   │ + clinical docs    │
          │ meditrans      │   └──────────────────┘   └────────────────────┘
          │                │   ┌──────────────────┐
          │ users, roles,  │   │ Embedding model  │
          │ sessions,      │◄──┤ bge-small-en-v1.5│
          │ embeddings     │   │ :11434 (Ollama)  │
          └────────────────┘   └──────────────────┘
```

Four external dependencies, each independently swappable:

1. **PostgreSQL 17 + pgvector** — the system of record. Required; the backend
   exits at startup if it cannot connect.
2. **A chat LLM behind an OpenAI-compatible `/v1/chat/completions`** — powers the
   AI Assistance module. Three supported deployments (local Ollama, private GPU
   VM, hosted provider); see [AI-MODELS.md](AI-MODELS.md).
3. **An embedding model behind an OpenAI-compatible `/v1/embeddings`** — powers
   semantic search. Must produce 384-dimension vectors.
4. **Corti cloud API** — speech-to-text and clinical document generation.
   Requires credentials; everything except transcription works without them.

---

## 2. Prerequisites

| Tool | Minimum | Verified on | Needed for |
|------|---------|-------------|------------|
| Go | 1.25 | 1.26.4 (darwin/arm64) | backend |
| Node.js | 20 | 22.16.0 | frontend |
| npm | 10 | 10.9.2 | frontend |
| PostgreSQL | 17 | 17.10 (Homebrew) | database |
| pgvector | 0.8 | 0.8.6 | embeddings, semantic search |
| Ollama | 0.10 | 0.30.11 | local models (skip if using a hosted API) |
| git, curl, jq | any recent | — | setup and verification |

### macOS (Homebrew)

```bash
brew install go node postgresql@17 pgvector ollama jq
brew link --force postgresql@17   # puts psql, pg_dump, pg_isready on PATH
```

`brew install ollama` installs the CLI and background service. The Ollama.app
desktop build works equally well — either way it must be listening on `:11434`.

### Ubuntu / Debian

```bash
sudo apt update
sudo apt install -y golang-go nodejs npm postgresql-17 postgresql-17-pgvector jq curl
curl -fsSL https://ollama.com/install.sh | sh
```

Check that `go version` reports 1.25 or newer; distro packages are often older,
in which case install from <https://go.dev/dl/>.

### Windows

Use WSL2 with the Ubuntu instructions. The helper scripts under `scripts/` and
`deploy/ai-vm/` are bash and assume a POSIX environment.

---

## 3. Clone the repository

```bash
git clone <repository-url> medical-transcription
cd medical-transcription
```

Layout:

```
medical-transcription/
├── backend/           Go + Fiber API server
│   ├── cmd/server/    entry point; also holds the legacy JSON seed data
│   ├── cmd/migrate-json/  one-time JSON → Postgres import
│   ├── cmd/mktoken/   mints a test JWT for API calls
│   └── internal/      ai, auth, corti, embedding, handlers, middleware, pgstore, utils
├── frontend/          Vue 3 + Vuetify SPA (Vite)
├── deploy/ai-vm/      GPU model server: Modelfile, setup.sh, provision_gcp.sh, manage.sh
├── scripts/           backup_db.sh, restore_db.sh
├── docs/              this guide and its companions
└── docker-compose.yml PostgreSQL + pgvector container (portable DB path)
```

---

## 4. Step 1 — PostgreSQL + pgvector

The backend applies its schema at every startup, so no migration tool is needed.
You only have to create an empty database with the `vector` extension available.

Pick **one** of the three options.

### Option A — Homebrew (macOS, no Docker)

```bash
brew services start postgresql@17
createdb meditrans
psql -d meditrans -c 'CREATE EXTENSION IF NOT EXISTS vector;'
```

Connection string: `postgres://localhost:5432/meditrans?sslmode=disable`
(no user or password — PostgreSQL authenticates as your OS user).

### Option B — Docker

```bash
docker compose up -d          # uses docker-compose.yml at the repo root
```

That starts `pgvector/pgvector:pg17` with database, user, and password all set
to `meditrans`, on port 5432, with a named volume `pgdata`.

Connection string:
`postgres://meditrans:meditrans@localhost:5432/meditrans?sslmode=disable`

### Option C — Linux service

```bash
sudo systemctl start postgresql
sudo -u postgres createuser --superuser "$USER"
createdb meditrans
psql -d meditrans -c 'CREATE EXTENSION IF NOT EXISTS vector;'
```

### Verify

```bash
pg_isready -h 127.0.0.1 -p 5432 -d meditrans
psql -d meditrans -Atc "SELECT extname, extversion FROM pg_extension WHERE extname = 'vector';"
```

Expected: `accepting connections`, then `vector|0.8.6` (any 0.8.x is fine). If
`CREATE EXTENSION` fails with *"extension vector is not available"*, the pgvector
package is not installed for **this** PostgreSQL major version — install it
(`brew install pgvector`, `apt install postgresql-17-pgvector`) and retry.

Full schema reference and the backup procedure are in [DATABASE.md](DATABASE.md).

---

## 5. Step 2 — AI models

Two separate models are needed. They are configured independently because a
hosted chat provider usually does not serve embedding models.

### 5.1 Embedding model — required for semantic search

```bash
ollama pull hf.co/CompendiumLabs/bge-small-en-v1.5-gguf
```

~24 MB. Verify it returns **384** dimensions — every `vector(384)` column in the
schema depends on this exact number:

```bash
curl -s http://localhost:11434/v1/embeddings \
  -H 'Content-Type: application/json' \
  -d '{"model":"hf.co/CompendiumLabs/bge-small-en-v1.5-gguf","input":["chest pain"]}' \
  | jq '.data[0].embedding | length'
```

Expected output: `384`. See [adr/0002](adr/0002-bge-small-384-not-368.md) for why
384 and not the "368" in the original task list.

### 5.2 Chat model — required for AI Assistance

Choose the deployment that matches the machine. All three speak the same
protocol; only `AI_BASE_URL`, `AI_MODEL`, and `AI_API_KEY` change.

**Local Ollama** (laptop development, no cost, ~2–2.5 min per extraction):

```bash
ollama create qwythos-16k-chat -f deploy/ai-vm/Modelfile
```

> **Do not** use `ollama pull hf.co/empero-ai/Qwythos-9B-...` or the older
> `qwythos-16k` tag directly. Ollama fails to convert the GGUF's chat template
> and silently falls back to `TEMPLATE {{ .Prompt }}`, which **drops system
> messages** — grounding and extraction then fail in ways that look like model
> quality problems rather than configuration problems. The `Modelfile` restores
> the native ChatML template and raises the context window to 16k.

The build takes a few minutes and produces a ~6.8 GB model. Verify:

```bash
ollama list | grep qwythos-16k-chat
```

**Private GPU VM** (fast, billed hourly) — see
[deploy/ai-vm/README.md](../deploy/ai-vm/README.md) and
[AI-MODELS.md § 6](AI-MODELS.md#6-deployment-options-for-the-chat-model):

```bash
bash deploy/ai-vm/provision_gcp.sh      # one-time: creates the VM and configures it
bash deploy/ai-vm/manage.sh start       # daily: starts VM + SSH tunnel on :8443
bash deploy/ai-vm/manage.sh stop        # stops GPU billing
```

**Hosted OpenAI-compatible provider** — nothing to install; set the three env
vars in the next step to the provider's endpoint, model name, and API key.

---

## 6. Step 3 — Backend configuration

```bash
cd backend
cp .env.example .env
```

Then edit `backend/.env`. Variables, where each is read, and what happens if it
is missing:

For AI Assistance chat and summarization, choose either:

- **Local Ollama:** private and free; configure `AI_BASE_URL` and `AI_MODEL` for
  the locally built chat model.
- **Hosted live-demo option:** use `AI_BASE_URL=https://api.llm-token.cn/v1`
  and `AI_MODEL=deepseek-v4-flash` when Ollama is not set up. The checked-in
  example includes a temporary, limited demo API key so the chat functionality
  works after cloning and copying `.env.example`. Rotate or remove it after the
  live demo.

This choice covers AI Assistance chat/summarization; semantic search still uses
the separate `EMBED_*` configuration. Process environment variables override
`.env`, so a managed secret can replace the demo key.

For the hosted option, choose the closest provider node when outside mainland
China:

| Region | `AI_BASE_URL` |
|--------|---------------|
| Mainland China | `https://api.llm-token.cn/v1` |
| Americas | `https://gpt-agent.cc/v1` |
| Europe / Africa | `https://eu.gpt-agent.cc/v1` |
| Southeast Asia, Hong Kong, Taiwan, Japan, Korea, Australia, New Zealand | `https://hk.gpt-agent.cc/v1` |

### Required

| Variable | Purpose | If unset |
|----------|---------|----------|
| `JWT_SECRET` | Signs login tokens | Falls back to a hardcoded demo secret and logs a warning — **never acceptable outside a laptop** |
| `DATABASE_URL` | PostgreSQL connection | Defaults to `postgres://localhost:5432/meditrans?sslmode=disable` |
| `SERVER_PORT` | Backend listen port | Defaults to `8080`; this project uses **8090** (see the port note below) |

Generate a secret with `openssl rand -base64 48`.

### Restore the supplied database backup (new machine)

If you received the three files for the demo backup, copy them into the
repository's `backups/` directory:

```text
meditrans-20260812-215036.dump
meditrans-20260812-215036.sql
meditrans-20260812-215036.manifest.txt
```

The `.dump` file is the preferred restore input. The `.sql` file is a plain-SQL
alternative; do not restore both. The manifest is not imported — it records the
expected row counts and PostgreSQL/pgvector versions for verification.

Stop the backend before restoring, then run this from the repository root:

```bash
bash scripts/restore_db.sh backups/meditrans-20260812-215036.dump
```

The script drops and recreates the database named by `DATABASE_URL`, asks for
confirmation, restores the schema and data, and prints the resulting row
counts. Use `--force` only when you are certain the target database may be
replaced:

```bash
bash scripts/restore_db.sh --force backups/meditrans-20260812-215036.dump
```

Check the manifest after the restore:

```bash
cat backups/meditrans-20260812-215036.manifest.txt
```

The supplied backup should contain 4 users, 3 roles, 6 sessions, 6 transcript
chunks, and 2 Corti templates. Full restore options and the manual `pg_restore`
equivalent are in [DATABASE.md § 8](DATABASE.md#8-restore). The backup contains
transcripts, AI extractions, and password hashes, so transfer it securely.

### Corti API — required for transcription only

| Variable | Purpose |
|----------|---------|
| `CORTI_CLIENT_ID`, `CORTI_CLIENT_SECRET` | OAuth2 client credentials from <https://console.corti.app> |
| `CORTI_TENANT`, `CORTI_ENVIRONMENT` | Tenant and region (`base`, `us`) |
| `CORTI_AUTH_URL`, `CORTI_API_BASE_URL`, `CORTI_WS_URL` | Endpoints; the defaults in `.env.example` match the US region |
| `CORTI_EMBEDDED_CLIENT_ID`, `CORTI_EMBEDDED_USERNAME`, `CORTI_EMBEDDED_PASSWORD` | Only for the `/embedded-assistant` page; without them that one endpoint returns 500 |

Without Corti credentials the app still runs: login, saved sessions, semantic
search, and the AI Assistance module all work on existing or pasted transcripts.
Only live recording and file transcription fail.

### AI Assistance (chat LLM)

| Variable | Live-demo example | Notes |
|----------|---------|-------|
| `AI_BASE_URL` | `https://api.llm-token.cn/v1` | Use the nearest regional endpoint above; local Ollama remains a documented fallback |
| `AI_MODEL` | `deepseek-v4-flash` | Low-cost DeepSeek v4 model listed by the provider; must match a model the endpoint serves |
| `AI_API_KEY` | `replace_with_provider_key` | Required for hosted providers; keep it in an environment secret or ignored `.env` |
| `AI_TEMPERATURE` | `0.2` | Stable sampling for the structured extraction; clamped to 0–2 |
| `AI_TIMEOUT_SECONDS` | `300` | Hosted extraction should finish well within this; local CPU inference may need the full timeout |
| `AI_MAX_COMPLETION_TOKENS` | `4096` | Covers the 15-field extraction and leaves room for provider reasoning tokens |

### Embeddings (semantic search)

| Variable | Default | Notes |
|----------|---------|-------|
| `EMBED_BASE_URL` | `http://localhost:11434/v1` | Configured separately from `AI_BASE_URL` on purpose |
| `EMBED_MODEL` | `hf.co/CompendiumLabs/bge-small-en-v1.5-gguf` | Must emit 384 dims |
| `EMBED_API_KEY` | empty | |
| `EMBED_TIMEOUT_SECONDS` | `60` | |

### Server and CORS

| Variable | Default | Notes |
|----------|---------|-------|
| `SERVER_HOST` | `0.0.0.0` | |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173,http://localhost:3000` | Comma-separated; must include the frontend origin |
| `LOG_LEVEL` | `debug` | |
| `TOKEN_REFRESH_BUFFER` | `60` | Seconds before Corti token expiry to refresh |

### The port note — read this one

The backend listens on **8090** in this project, not the 8080 in the older docs:
port 8080 is commonly occupied (Open WebUI, other local services). Vite's
built-in fallback proxy target is `http://localhost:8080`, so if you leave
`SERVER_PORT=8090` and skip the frontend env file in the next step, **every API
call 404s or hangs with no obvious error**. Set both, or set neither.

### A note on `.env` loading

`utils.LoadConfig` searches `.env`, `backend/.env`, `../backend/.env`,
`../../.env` in that order, takes the first one found, and fills only variables
that are not already present in the process environment. A shell-exported value
therefore overrides the file — useful for injecting `AI_API_KEY` on an
instructor or CI machine.

---

## 7. Step 4 — Frontend configuration

```bash
cd frontend
cp .env.example .env.local
npm install
```

`.env.local` needs one variable, matching `SERVER_PORT` from the backend:

```env
VITE_BACKEND_URL=http://localhost:8090
```

Vite proxies `/api/*` (including WebSocket upgrades) to that target. `.env.local`
is gitignored, so **every machine must create it** — this is the single most
common reason a fresh clone appears broken.

---

## 8. Step 5 — Run it

Two terminals, backend first.

```bash
# Terminal 1
cd backend
go mod download
go run ./cmd/server
```

A healthy startup logs, in order:

```
Loaded .env from: .env
Embedding pipeline: endpoint=http://localhost:11434/v1 model=hf.co/CompendiumLabs/bge-small-en-v1.5-gguf dims=384
embedding worker: started (model=..., 384 dims)
AI Assistance module: endpoint=... model=... temperature=0.6 max_completion_tokens=4096
Routes configured (RBAC v2 — permission-based)
Starting Corti Backend Server on 0.0.0.0:8090
```

On a fresh database it also logs `pgstore: seeded default system roles` and
`pgstore: seeded default superuser`.

```bash
# Terminal 2
cd frontend
npm run dev
```

Open <http://localhost:5173>. The seeded superuser on a fresh database is:

```
admin@xstek.net / super@xstek2026
```

**Change this password immediately on any machine that is not a local laptop.**
It is seeded in `internal/pgstore/pgstore.go` only when the `users` table is
empty, so it never overwrites a real account.

---

## 9. Step 6 — Verify the installation

Run these in order. Each one isolates a different layer, so the first failure
tells you which step to revisit.

```bash
# 1. Backend is up
curl -s http://localhost:8090/health | jq
# → {"service":"corti-backend","status":"healthy","version":"2.0.0"}

# 2. Frontend is up and proxying
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:5173/
# → 200

# 3. Database is reachable and seeded
psql -d meditrans -Atc 'SELECT count(*) FROM roles;'
# → 3 or more

# 4. Mint a test token (from backend/)
cd backend
JWT=$(JWT_SECRET=$(grep '^JWT_SECRET=' .env | cut -d= -f2) go run ./cmd/mktoken)

# 5. Chat LLM is reachable and the model exists
curl -s http://localhost:8090/api/ai/status -H "Authorization: Bearer $JWT" | jq
# → {"success":true,"status":{"reachable":true,"model_available":true,...}}

# 6. Semantic search end-to-end (embeds the query, runs cosine search)
curl -s -X POST http://localhost:8090/api/search \
  -H "Authorization: Bearer $JWT" -H 'Content-Type: application/json' \
  -d '{"query":"chest pain"}' | jq
# → {"success":true,"results":[...]}   (empty results on a fresh database is correct)
```

`mktoken` embeds `ambient.access` and a `superuser` role, and defaults to user ID
`e2e-test`. Search results are scoped to the token's user ID, so pass
`-user <real-user-id>` to see a real account's sessions.

### Automated checks

```bash
cd backend  && go build ./... && go test ./...   # all packages pass
cd frontend && npm test                          # vitest, 41 tests
cd frontend && npm run build                     # vue-tsc type-check + bundle
```

`npm run lint` is listed in `package.json` but **eslint is not in
`devDependencies`** — the script has never worked. A failure there is
pre-existing, not something you broke.

---

## 10. Optional — import the legacy JSON data

Before the PostgreSQL migration ([adr/0001](adr/0001-postgres-pgvector-replaces-json-storage.md)),
users, roles, sessions, and templates lived in JSON files. They are kept as an
import source. To load them into a fresh database:

```bash
cd backend
go run ./cmd/migrate-json                        # reads ./data + ./templates/custom
go run ./cmd/migrate-json -data cmd/server/data  # the other data copy
```

The import is idempotent (everything is upserted) and does **not** compute
embeddings — the backend's embedding worker backfills them within 60 seconds of
the next startup.

---

## 11. Optional — one-command start/stop

A controller script wraps the whole stack: database, GPU VM, SSH tunnel,
backend, frontend, plus a verification pass. On this project it is installed as
an agent skill at `~/.codex/skills/manage-medical-transcription/`.

```bash
SKILL_DIR="${CODEX_HOME:-$HOME/.codex}/skills/manage-medical-transcription"
bash "$SKILL_DIR/scripts/project_ctl.sh" status
```

| Command | Effect |
|---------|--------|
| `start` | Ensures PostgreSQL, starts the VM + tunnel (only if `AI_BASE_URL` points at the tunnel), builds and starts backend and frontend, then verifies the full path |
| `stop` | Stops frontend, backend, tunnel, VM; stops PostgreSQL **only if it started it** |
| `restart` | `stop` then `start` |
| `status` | Reports every port, PostgreSQL, pgAdmin, and VM state without starting anything |
| `verify` | Checks the running frontend → backend → model path |
| `logs` | Tails the backend and frontend logs |
| `open-pgadmin` | Opens pgAdmin (never automatic) |
| `stop-all` | `stop` plus the local Open WebUI and Ollama launch services |

### Running it on a different machine

The script hardcodes this machine's defaults. Override them with environment
variables rather than editing it:

```bash
MEDICAL_TRANSCRIPTION_ROOT=/path/to/medical-transcription \
MEDICAL_TRANSCRIPTION_BACKEND_PORT=8090 \
MEDICAL_TRANSCRIPTION_FRONTEND_PORT=5173 \
MEDICAL_TRANSCRIPTION_DB_MODE=homebrew \
  bash "$SKILL_DIR/scripts/project_ctl.sh" start
```

| Variable | Default |
|----------|---------|
| `MEDICAL_TRANSCRIPTION_ROOT` | `/Users/yitaowang/Downloads/Yitao_Wang/medical-transcription` |
| `MEDICAL_TRANSCRIPTION_STATE_DIR` | `~/.cache/manage-medical-transcription` |
| `MEDICAL_TRANSCRIPTION_BACKEND_PORT` / `_FRONTEND_PORT` / `_TUNNEL_PORT` | `8090` / `5173` / `8443` |
| `MEDICAL_TRANSCRIPTION_DB_HOST` / `_DB_PORT` / `_DB_NAME` | `127.0.0.1` / `5432` / `meditrans` |
| `MEDICAL_TRANSCRIPTION_DB_MODE` | `auto` (`homebrew` \| `docker` \| `external` \| `auto`) |
| `MEDICAL_TRANSCRIPTION_VM_INSTANCE` / `_VM_ZONE` | `ai-model-server` / auto-discovered |
| `MEDICAL_TRANSCRIPTION_PGADMIN_APP` | `/Applications/pgAdmin 4.app` |

Before it will start, the controller enforces:

- `SERVER_PORT` in `backend/.env` equals the backend port (8090)
- `VITE_BACKEND_URL` in `frontend/.env.local` equals `http://localhost:8090`
- `AI_BASE_URL` is either the local tunnel or an `https://` endpoint —
  a plain-HTTP remote endpoint is rejected, since transcripts must not cross the
  internet unencrypted
- `AI_MODEL` and `AI_API_KEY` are non-empty

Required commands: `go`, `npm`, `node`, `curl`, `jq`, `lsof`, `rsync`, `rg`
(ripgrep), and `gcloud`. Two portability caveats:

- `rg` is used to read Homebrew service state but is never checked for; without
  it the script treats a running PostgreSQL as stopped.
- `stop` requires `gcloud` unconditionally, even when the AI endpoint is hosted
  and no VM exists. On a machine without `gcloud`, stop the processes manually.

On macOS the controller runs backend and frontend under `launchctl` from copies
in its state directory, syncing `data/` and `templates/` back to the repo on
stop. On other platforms it uses plain `nohup`.

---

## 12. Production notes

```bash
# Backend — static binary
cd backend
go build -o corti-backend ./cmd/server
./corti-backend

# Frontend — static bundle in frontend/dist/, serve behind nginx or similar
cd frontend
npm run build
```

Checklist before exposing anything publicly:

- [ ] `JWT_SECRET` set to a generated value, not the fallback
- [ ] Seeded superuser password changed
- [ ] `CORS_ALLOWED_ORIGINS` restricted to the real frontend origin
- [ ] `DATABASE_URL` uses a dedicated role with a password and `sslmode=require`
- [ ] The model endpoint is not plain HTTP across a public network — use a VPC,
      SSH tunnel, WireGuard/Tailscale, or TLS in front of nginx
      (see [deploy/ai-vm/README.md](../deploy/ai-vm/README.md#security--read-before-exposing-anything))
- [ ] Backups scheduled — [DATABASE.md § 7](DATABASE.md#7-backup)
- [ ] `LOG_LEVEL` lowered from `debug`

---

## 13. Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Backend exits: `Failed to connect to Postgres` | Server down or database missing | `brew services start postgresql@17 && createdb meditrans` |
| `Failed to apply database schema: ... type "vector" does not exist` | pgvector not installed for this PG major version | `brew install pgvector` / `apt install postgresql-17-pgvector`, then restart PostgreSQL |
| Frontend loads, every API call fails | `frontend/.env.local` missing, so Vite proxies to :8080 | Create it with `VITE_BACKEND_URL=http://localhost:8090` |
| `address already in use` on :8080 | Open WebUI or another service owns it | This project uses 8090; check `lsof -tiTCP:8080 -sTCP:LISTEN` |
| `/api/ai/status` → `reachable:false` | Model endpoint down or wrong URL | `curl $AI_BASE_URL/models`; start Ollama or the VM tunnel |
| `/api/ai/status` → `model_available:false` | `AI_MODEL` does not match a served model | `ollama list`, then align `AI_MODEL` |
| AI answers ignore the transcript; extraction is nonsense | Wrong Ollama model build — broken `{{ .Prompt }}` template drops system messages | Rebuild: `ollama create qwythos-16k-chat -f deploy/ai-vm/Modelfile`. Never use the raw HF pull or the `qwythos-16k` tag |
| Extraction returns truncated JSON | `AI_MAX_COMPLETION_TOKENS` too low | Set 4096 or higher |
| Extraction times out | Local CPU inference is slow | Raise `AI_TIMEOUT_SECONDS`, or move to the GPU VM |
| Model replies "I am Qwythos…" or emits only reasoning | Fine-tune artifacts | Already stripped and retried once in `internal/ai/openai_compat.go`; keep temperature ≥ 0.6 |
| Search returns nothing on a populated database | Embeddings still pending, or the embedder is down | `psql -d meditrans -Atc 'SELECT count(*) FROM sessions WHERE embedded_at IS NULL;'`; check the embedding endpoint and the backend log |
| `expected 384 dimensions, got N` | Wrong embedding model | Only `bge-small-en-v1.5` (384) matches the schema; changing models means re-embedding everything |
| Shell env var seems ignored | `.env` overrides the shell environment | Edit `backend/.env`, or run from a directory with its own `.env` |
| `npm run lint` fails | eslint is missing from `devDependencies` | Pre-existing; use `npm run build` for type checking |
| Controller says PostgreSQL is stopped when it is running | `rg` (ripgrep) not installed | `brew install ripgrep` |
| `ZONE_RESOURCE_POOL_EXHAUSTED` when starting the VM | No L4 capacity in that zone | The scripts fail over across zones automatically; or set `ZONES="..."` |

---

## 14. Verified configuration

This guide was validated end to end on:

| Component | Version |
|-----------|---------|
| macOS | Darwin 24.3.0 (Apple Silicon) |
| Go | 1.26.4 |
| Node.js / npm | 22.16.0 / 10.9.2 |
| PostgreSQL | 17.10 (Homebrew) |
| pgvector | 0.8.6 |
| Ollama | 0.30.11 |
| Chat model | `qwythos-16k-chat` (Qwythos-9B Q4_K_M, 16k ctx) |
| Embedding model | `hf.co/CompendiumLabs/bge-small-en-v1.5-gguf` (384 dims) |

Backend `go build ./... && go test ./...`: all packages pass.
Frontend `npm test`: 41 tests pass.
