# xstek Medical Transcription

A production-grade full-stack application for medical transcription powered by advanced AI speech recognition and ambient AI technology.

![Vue.js](https://img.shields.io/badge/Vue.js-3.x-4FC08D?style=flat&logo=vue.js)
![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)
![Vuetify](https://img.shields.io/badge/Vuetify-3.x-1867C0?style=flat&logo=vuetify)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17%20%2B%20pgvector-336791?style=flat&logo=postgresql)
![License](https://img.shields.io/badge/License-MIT-green.svg)

## ✨ Features

### 📁 Async File Transcription
- Upload audio files (WAV, MP3, M4A, FLAC, OGG, WebM)
- Automatic language detection
- Speaker diarization
- Structured transcript with timestamps
- Clinical document generation

### 🎙️ Real-Time Ambient Streaming
- Live microphone capture
- PCM16 audio streaming (16kHz, mono)
- Real-time partial and final transcripts
- Medical entity detection
- Session summary export

### 🧠 AI Assistance
- 15-field structured clinical extraction from a completed transcript
- Q&A grounded strictly in the consultation record
- PDF / Word / Markdown report export
- Provider-agnostic: local Ollama, a private GPU VM, or a hosted API

### 🔍 Semantic Search
- Search your own sessions by meaning, not keywords
- 384-dim `bge-small-en-v1.5` embeddings stored in pgvector
- Computed asynchronously — a save never waits on the model

## 🏗️ Architecture

```
medical-transcription/
├── backend/                 # Go + Fiber API server
│   ├── cmd/server/         # Application entry point
│   ├── cmd/migrate-json/   # One-time JSON → Postgres import
│   ├── cmd/mktoken/        # Test JWT minting
│   └── internal/
│       ├── ai/             # LLM + embedding clients, prompts, chunker
│       ├── auth/           # JWT, RBAC, storage contracts
│       ├── corti/          # Corti API clients and WebSocket proxies
│       ├── embedding/      # Async embedding worker
│       ├── handlers/       # HTTP handlers, one per feature
│       ├── middleware/     # CORS, logging, recovery, permissions
│       ├── pgstore/        # PostgreSQL + pgvector stores + schema.sql
│       └── utils/          # Audio validation, env loading
├── frontend/               # Vue 3 + Vuetify SPA (Vite)
│   └── src/                # views, components, composables, stores, services
├── deploy/ai-vm/           # GPU model server: Modelfile, setup, provisioning
├── scripts/                # Database backup and restore
├── docs/                   # Setup, database, AI models, ADRs
└── docker-compose.yml      # PostgreSQL + pgvector container
```

## 🚀 Quick Start

**Full instructions: [docs/SETUP.md](docs/SETUP.md).** A working install needs
PostgreSQL + pgvector and two AI models as well as the two servers below; the
short version is here, the details and every troubleshooting case are there.

### Prerequisites

- **Go** 1.25+ · **Node.js** 20+ · **PostgreSQL** 17 with **pgvector**
- **Ollama** (for local AI models) — or a hosted OpenAI-compatible endpoint
- **Corti API credentials** (only for transcription features)

### 1. Database

```bash
brew services start postgresql@17 && createdb meditrans
psql -d meditrans -c 'CREATE EXTENSION IF NOT EXISTS vector;'
```

### 2. AI models

```bash
ollama pull hf.co/CompendiumLabs/bge-small-en-v1.5-gguf   # embeddings, 384 dims
ollama create qwythos-16k-chat -f deploy/ai-vm/Modelfile  # chat model
```

Build the chat model from the Modelfile — a direct `ollama pull` of the GGUF
imports a broken chat template that silently drops system messages. See
[docs/AI-MODELS.md](docs/AI-MODELS.md).

### 3. Configure

```bash
cp backend/.env.example backend/.env       # credentials, JWT_SECRET, model endpoints
cp frontend/.env.example frontend/.env.local
```

### 4. Run

```bash
cd backend  && go mod download && go run ./cmd/server   # http://localhost:8090
cd frontend && npm install && npm run dev               # http://localhost:5173
```

Default login on a fresh database: `admin@xstek.net` / `super@xstek2026` —
change it immediately outside local development.

## 📡 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/auth/login` | Login |
| `POST` | `/api/transcribe/upload` | Upload audio file |
| `GET` | `/api/transcribe/:id` | Get transcript |
| `GET` | `/api/transcribe/:id/poll` | Long-poll for completion |
| `POST` | `/api/ambient/start` | Start ambient session |
| `WS` | `/api/ambient/ws/:id` | WebSocket streaming |
| `POST` | `/api/ai/summarize` | 15-field clinical extraction from a transcript |
| `POST` | `/api/ai/chat` | Q&A grounded in the extraction or transcript |
| `GET` | `/api/ai/status` | Model endpoint reachability |
| `POST` | `/api/search` | Semantic search over your own sessions |
| `GET`/`POST` | `/api/sessions` | Saved sessions |
| `GET` | `/health` | Health check |

## 🔧 Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `CORTI_CLIENT_ID` / `CORTI_CLIENT_SECRET` | Corti API credentials | Required for transcription |
| `JWT_SECRET` | Signs login tokens | Falls back to a demo secret |
| `DATABASE_URL` | PostgreSQL + pgvector | `postgres://localhost:5432/meditrans?sslmode=disable` |
| `SERVER_PORT` | Backend port | `8090` |
| `CORS_ALLOWED_ORIGINS` | CORS origins | `http://localhost:5173` |
| `AI_BASE_URL` / `AI_MODEL` / `AI_API_KEY` | Chat model endpoint | Hosted live demo: `https://api.llm-token.cn/v1`, `deepseek-v4-flash` |
| `EMBED_BASE_URL` / `EMBED_MODEL` | Embedding endpoint (384 dims) | `http://localhost:11434/v1`, `bge-small-en-v1.5` |

Every variable is documented in [docs/SETUP.md § 6](docs/SETUP.md#6-step-3--backend-configuration).

## 🎨 Screenshots

The application features a modern dark theme with:
- Real-time waveform visualization
- Live transcript display
- Medical event detection cards
- Drag & drop file upload

## 📚 Documentation

| Document | Covers |
|----------|--------|
| [Setup Guide](docs/SETUP.md) | Full install on a clean machine, verification, troubleshooting |
| [Database Guide](docs/DATABASE.md) | Schema, embedding lifecycle, backup and restore |
| [AI Model Dependencies](docs/AI-MODELS.md) | Every model and service, deployment options, failure modes |
| [Optimization Review](docs/OPTIMIZATION.md) | Measured findings and a prioritized improvement list |
| [API Reference](docs/API.md) | Endpoint contracts |
| [Architecture Decisions](docs/adr/) | Why Postgres+pgvector, 384-dim embeddings, hand-written SQL |
| [Model Server Deployment](deploy/ai-vm/README.md) | GPU VM provisioning, tunnel, security |

## 🔒 Security

- OAuth2 tokens are cached and auto-refreshed server-side
- Corti credentials never reach the frontend
- All audio files are validated for type and size
- CORS properly configured

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please read our contributing guidelines before submitting PRs.

---

Built with ❤️ by xstek
