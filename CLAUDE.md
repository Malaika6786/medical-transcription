# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Corti Medical Transcription Demo — a full-stack application for AI-powered medical transcription. Monorepo with a Go backend and Vue 3 frontend. The backend proxies all Corti API calls so credentials never reach the browser.

## Development Commands

### Backend (Go + Fiber)
```bash
cd backend
go mod download
go run cmd/server/main.go          # Start server on :8080
go build -o corti-backend cmd/server/main.go
```

### Frontend (Vue 3 + Vite)
```bash
cd frontend
npm install
npm run dev        # Dev server on :5173 (proxies /api → localhost:8080)
npm run build      # Type-check + Vite bundle
npm run lint       # ESLint auto-fix
npm run preview    # Preview production build
```

Both servers must run simultaneously for full functionality. No tests are currently configured.

## Architecture

### Backend (`backend/`)

Entry point: `cmd/server/main.go` — initializes Fiber, registers middleware and routes, loads `.env`, creates four Corti API clients.

```
internal/
├── ai/            # AI Assistance module: provider-agnostic LLM client (OpenAI-compatible), prompts, summary parsing, embeddings client + chunker (embed.go)
├── auth/          # JWT auth, RBAC models, storage contracts (storage.go) — implemented by internal/pgstore
├── corti/         # Corti API clients: token_manager, async_client, ambient_proxy, dictation_proxy
├── embedding/     # Async embedding worker: backfills/refreshes vectors, notified on session save
├── handlers/      # One handler file per feature (auth, async, ambient, dictation, sessions, ai, search)
├── middleware/    # CORS, request logger (with requestID), panic recovery
├── pgstore/       # PostgreSQL + pgvector stores (pgx, hand-written SQL) + embedded schema.sql
└── utils/         # Audio validation, env loading
```

**Storage**: PostgreSQL + pgvector is the system of record (users, roles, sessions, Corti templates, embeddings) — see `docs/adr/0001..0003` and `CONTEXT.md` for the domain glossary. `DATABASE_URL` configures it (local default `postgres://localhost:5432/meditrans`; `docker-compose.yml` at the repo root for Docker setups). The JSON files in `cmd/server/data/` are legacy — imported once via `go run ./cmd/migrate-json` as the one-time import source; the old JSON-file store implementations themselves have been removed (dead code cleanup), so `cmd/migrate-json` reads the files directly rather than through a `UserStore`/`RoleStore`/`SessionStore`. Embeddings are 384-dim `bge-small-en-v1.5` served by Ollama (`EMBED_*` env vars), computed asynchronously: session saves notify `internal/embedding.Worker`, NULL vectors are backfilled at startup, and `POST /api/search` runs cosine search over transcript chunks ∪ extraction embeddings scoped to the caller's own sessions.

**Key pattern — Token Manager**: OAuth2 tokens are cached server-side with a 60-second refresh buffer. The backend acts as a WebSocket proxy between the browser and Corti API, injecting fresh auth tokens transparently.

**Corti API clients** (all initialized in `main.go`, passed to handlers):
- `TokenManager` — OAuth2 token caching/refresh
- `AsyncClient` — file upload transcription
- `AmbientProxy` — WebSocket proxy for live ambient streaming
- `DictationProxy` — WebSocket proxy for live dictation

### Frontend (`frontend/src/`)

```
views/         # Full-page components (DictationPage.vue is largest at ~86KB)
components/    # Reusable UI pieces (ReportGenerator, RichTextEditor, etc.)
composables/   # Logic extracted as Vue 3 composables
stores/        # Pinia: auth.ts (tokens, RBAC) + sessions.ts (cached sessions, max 5/user)
services/api.ts  # Axios instance with auth interceptors and session-expiry handling
router/index.ts  # Route guards use meta fields; roles enforced here and in App.vue
```

**Key composables**:
- `useAmbientSession.ts` — WebSocket lifecycle for live ambient recording
- `useAudioCapture.ts` — Microphone access via MediaRecorder
- `useDictation.ts` — Dictation session state, large and complex
- `useTranscription.ts` — File-based async transcription polling

**State management**: Pinia `auth` store holds the JWT, user data, and computed role permissions (`isSuperuser`, etc.). The `sessions` store manages saved sessions per user (no cap — removed at the user's request), differentiated by type (`ambient`, `file-transcription`, `dictation`).

### API Routes

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/auth/login` | Login |
| GET | `/api/auth/me` | Current user (protected) |
| POST | `/api/auth/force-logout-all` | Invalidate all tokens (superuser) |
| GET/POST/PUT/DELETE | `/api/users` | User management (superuser) |
| POST | `/api/transcribe/upload` | Upload audio → `interaction_id` |
| GET | `/api/transcribe/:id` | Get transcript |
| GET | `/api/transcribe/:id/poll` | Long-poll for completion |
| POST | `/api/transcribe/:id/document` | Generate clinical document |
| POST | `/api/ambient/start` | Start ambient session → `interaction_id` + WS URL |
| WS | `/api/ambient/ws/:id` | Bidirectional WebSocket (PCM16, 16kHz, mono) |
| POST | `/api/dictation/start` | Start dictation session |
| WS | `/api/dictation/ws` | Dictation WebSocket |
| POST | `/api/ai/summarize` | AI summary of a completed transcript (stateless — transcript in body) |
| POST | `/api/ai/chat` | Transcript-grounded Q&A with chat history |
| GET | `/api/ai/status` | LLM endpoint reachability + model availability |
| POST | `/api/search` | Semantic search over the caller's own sessions (query embedded via bge, cosine-ranked) |
| GET | `/health` | Health check |

### RBAC

Three roles: `superuser` > `doctor` > `user`. Permissions are enforced at the route guard level (frontend), component level (via auth store computed properties), and handler level (backend middleware checks JWT claims). Superusers manage users and can force-logout everyone.

## Environment Configuration

Full setup instructions, every variable, and the troubleshooting table live in `docs/SETUP.md`; database operations and backup/restore in `docs/DATABASE.md`; model dependencies in `docs/AI-MODELS.md`.

Copy `backend/.env.example` to `backend/.env`. Required variables:

```
CORTI_CLIENT_ID=
CORTI_CLIENT_SECRET=
CORTI_AUTH_URL=https://auth.us.corti.app/realms/base/protocol/openid-connect/token
CORTI_API_BASE_URL=https://api.us.corti.app
CORTI_WS_URL=wss://api.us.corti.app/stt/stream
SERVER_PORT=8090
CORS_ALLOWED_ORIGINS=http://localhost:5173
JWT_SECRET=
DATABASE_URL=postgres://localhost:5432/meditrans?sslmode=disable
```

The Vite dev server proxies `/api/*` to `VITE_BACKEND_URL` from `frontend/.env.local` (falling back to `localhost:8080`, which is wrong for this project — the backend listens on 8090). That file is gitignored, so every machine must create it from `frontend/.env.example`.

`utils.LoadConfig` loads the first `.env` it finds, while preserving explicit process environment values. This lets deployments inject secrets such as `AI_API_KEY` without editing the file.

The app is deployed to a real server — treat CORS config, `JWT_SECRET`, `DATABASE_URL`, and the model endpoint's transport security as production concerns.

There is no cap on saved sessions per user — the earlier 5-session limit was removed from both the backend (`pgstore.SaveSession`) and the frontend (`stores/sessions.ts`) at the user's request. There is no soft delete.

## Document Templates

Clinical document templates (GP letters, summaries) live in `backend/cmd/server/templates/custom/`. The `POST /api/transcribe/:id/document` endpoint uses these to generate structured clinical output from raw transcripts.

`CortiSectionsPage.vue` is the UI for browsing available Corti API sections. These sections are the building blocks for custom templates — users select sections to compose a template (e.g., a GP letter), which gets saved to the templates directory. Treat `CortiSectionsPage.vue` and the template system as a connected pair.

## AI Assistance Module

Post-recording clinical extraction + grounded Q&A, independent of Corti (the transcript text is the only handoff). Backend: `internal/ai/` (LLMClient interface + OpenAI-compatible implementation) and `handlers/ai_handler.go`, gated by `ambient.access`. Frontend: `composables/useAiAssistant.ts` + `components/AiAssistantPanel.vue`, shown via the "AI Assistance" button on `AmbientSessionPage.vue` after a recording ends.

`POST /api/ai/summarize` returns a nested 15-field `extraction` object (team schema, camelCase: `summary`, `chiefComplaint`, `history`, `allergies`, `medications`, `symptoms`, `diagnosis`, `differentialDiagnosis`, `treatment`, `labTests`, `procedures`, `followUp`, `riskFactors`, `medicalTerms`, `actionItems`). `POST /api/ai/chat` grounds in the stored `extraction` when the client sends one (preferred — far fewer prompt tokens than the transcript) and falls back to `transcript` grounding otherwise; the frontend stores the extraction after summarize and sends it with every question.

Persistence: after a successful summarize the frontend auto-saves the session via `POST /api/sessions` with the extraction attached (`SavedSession.Extraction`; on update, a nil extraction never overwrites a stored one — `extraction = COALESCE($6, extraction)`). Reopening a saved session shows an "AI Extraction" tab in the detail dialog and resumes Q&A grounded in the stored record. Note the posture change: the server persists extraction content (PHI) in the `sessions.extraction` JSONB column — a deliberate decision, previously the AI module was fully stateless. Chat history is still not persisted.

Config (all in `backend/.env`): the portable live-demo example uses `AI_BASE_URL=https://api.llm-token.cn/v1`, `AI_MODEL=deepseek-v4-flash`, `AI_API_KEY` supplied locally, `AI_TEMPERATURE=0.2`, `AI_TIMEOUT_SECONDS=300`, and `AI_MAX_COMPLETION_TOKENS=4096`. Use the regional Base URL table in `docs/SETUP.md` when the instructor is outside mainland China; local Ollama remains a fallback.

The model is built from `deploy/ai-vm/Modelfile` (`ollama create qwythos-16k-chat -f deploy/ai-vm/Modelfile`) — do NOT use the raw `hf.co/empero-ai/Qwythos-...` pull or the old `qwythos-16k` tag directly: Ollama imports them with a broken `{{ .Prompt }}` template that drops system messages and breaks grounding. `internal/ai/openai_compat.go` strips `<think>` reasoning traces and known identity artifacts server-side and retries once on empty output; keep temperature ≥ 0.6 (greedy sampling loops). GPU VM deployment: `deploy/ai-vm/README.md`. Mint a local test JWT with `JWT_SECRET=... go run ./cmd/mktoken`.
