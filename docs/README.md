# xstek Medical Transcription

A production-grade full-stack application for medical transcription with AI-powered speech recognition and ambient AI technology.

> **Installing the project? Use [SETUP.md](SETUP.md).**
> The "Getting Started" section below predates the PostgreSQL + pgvector
> migration and the AI Assistance module: it omits the database and both AI
> models, and it still names port 8080 (the backend now listens on 8090).
> This file is kept for its API, WebSocket, and audio-format reference.

## Architecture Overview

```
corti-medical-transcription-demo/
├── backend/                    # Go + Fiber backend
│   ├── cmd/server/main.go     # Application entry point
│   ├── internal/
│   │   ├── corti/             # Corti API clients
│   │   │   ├── types.go       # Type definitions
│   │   │   ├── token_manager.go # OAuth2 token management
│   │   │   ├── async_client.go  # Async transcription client
│   │   │   └── ambient_proxy.go # WebSocket proxy
│   │   ├── handlers/          # HTTP request handlers
│   │   ├── middleware/        # Fiber middleware
│   │   └── utils/            # Utility functions
│   ├── go.mod
│   └── .env.example
├── frontend/                  # Vue 3 + Vuetify frontend
│   ├── src/
│   │   ├── components/       # Vue components
│   │   ├── composables/      # Vue composables
│   │   ├── views/           # Page components
│   │   ├── router/          # Vue Router config
│   │   └── plugins/         # Vuetify config
│   └── package.json
└── docs/                     # Documentation
```

## Features

### Async Transcription (File Upload)
- Upload audio files (WAV, MP3, M4A, FLAC, OGG, WebM)
- Automatic language detection
- Speaker diarization
- Structured transcript with timestamps
- Clinical document generation

### Real-Time Ambient Streaming
- Live microphone capture
- PCM16 audio streaming (16kHz, mono)
- Real-time partial and final transcripts
- Medical entity detection
- Session summary export

## Getting Started

### Prerequisites
- Go 1.23+
- Node.js 20+
- Corti API credentials (from [Corti Console](https://console.corti.app))

### Backend Setup

1. Navigate to backend directory:
```bash
cd backend
```

2. Copy environment template:
```bash
cp .env.example .env
```

3. Edit `.env` with your Corti credentials:
```env
CORTI_CLIENT_ID=your_client_id_here
CORTI_CLIENT_SECRET=your_client_secret_here
```

4. Install dependencies:
```bash
go mod download
```

5. Run the server:
```bash
go run cmd/server/main.go
```

The backend will start on `http://localhost:8080`

### Frontend Setup

1. Navigate to frontend directory:
```bash
cd frontend
```

2. Install dependencies:
```bash
npm install
```

3. Start development server:
```bash
npm run dev
```

The frontend will start on `http://localhost:5173`

## API Endpoints

### Async Transcription

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/transcribe/upload` | Upload audio file for transcription |
| GET | `/api/transcribe/:interactionId` | Get transcript by interaction ID |
| GET | `/api/transcribe/:interactionId/poll` | Long-poll for transcript completion |
| POST | `/api/transcribe/:interactionId/document` | Generate clinical document |

### Ambient Streaming

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/ambient/start` | Start new ambient session |
| GET | `/api/ambient/session/:interactionId` | Get session status |
| GET | `/api/ambient/stats` | Get server statistics |
| WS | `/api/ambient/ws/:interactionId` | WebSocket for audio streaming |

## WebSocket Protocol

### Client → Server Messages

**Audio Data (Binary)**
- Raw PCM16 audio frames (16kHz, mono)

**Control Messages (JSON)**
```json
{ "type": "ping" }
{ "type": "stop" }
{ "type": "config", "data": { "language": "en-US" } }
```

### Server → Client Messages

```json
// Connection established
{ "type": "connected", "data": { "interaction_id": "..." }, "timestamp": 1234567890 }

// Transcript event
{ "type": "transcript", "data": { "text": "...", "is_final": true, "speaker": "Doctor" }, "timestamp": 1234567890 }

// Medical event detected
{ "type": "medical_event", "data": { "category": "medication", "value": "Aspirin 100mg" }, "timestamp": 1234567890 }

// Error
{ "type": "error", "error": "...", "timestamp": 1234567890 }
```

## Audio Requirements

For optimal transcription quality:

| Property | Value |
|----------|-------|
| Sample Rate | 16000 Hz |
| Channels | 1 (Mono) |
| Encoding | PCM16 (16-bit signed integer) |
| Bit Depth | 16 bits |

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `CORTI_CLIENT_ID` | Corti API client ID | Required |
| `CORTI_CLIENT_SECRET` | Corti API client secret | Required |
| `CORTI_AUTH_URL` | OAuth2 token endpoint | `https://auth.corti.app/oauth/token` |
| `CORTI_API_BASE_URL` | Corti API base URL | `https://api.corti.app` |
| `CORTI_WS_URL` | Corti WebSocket URL | `wss://api.corti.app/stt/stream` |
| `SERVER_PORT` | Backend server port | `8080` |
| `SERVER_HOST` | Backend server host | `0.0.0.0` |
| `CORS_ALLOWED_ORIGINS` | CORS allowed origins | `http://localhost:5173` |
| `LOG_LEVEL` | Logging level | `debug` |
| `TOKEN_REFRESH_BUFFER` | Seconds before token expiry to refresh | `60` |

## Security Considerations

1. **Token Management**: OAuth2 tokens are cached and auto-refreshed server-side
2. **No Client Exposure**: Corti credentials never reach the frontend
3. **CORS Configuration**: Configure allowed origins for production
4. **Audio Validation**: All uploaded files are validated for type and size
5. **Rate Limiting**: Consider adding rate limiting for production

## Production Deployment

### Backend

```bash
# Build binary
cd backend
go build -o corti-backend cmd/server/main.go

# Run with production settings
export GIN_MODE=release
./corti-backend
```

### Frontend

```bash
# Build for production
cd frontend
npm run build

# Serve with nginx or similar
```

### Docker (Example)

```dockerfile
# Backend Dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN go build -o main cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/main /main
CMD ["/main"]
```

## Troubleshooting

### Common Issues

1. **"Missing Corti client credentials"**
   - Ensure `CORTI_CLIENT_ID` and `CORTI_CLIENT_SECRET` are set

2. **WebSocket connection fails**
   - Check CORS configuration
   - Verify interaction ID is valid

3. **Audio not capturing**
   - Check browser microphone permissions
   - Verify correct audio device is selected

4. **Transcript not appearing**
   - Check browser console for WebSocket errors
   - Verify audio is being captured (check waveform)

## License

MIT License - See LICENSE file for details.

