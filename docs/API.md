# API Documentation

## Async Transcription API

### Upload Audio for Transcription

**POST** `/api/transcribe/upload`

Upload an audio file to initiate transcription.

**Request:**
- Content-Type: `multipart/form-data`
- Body:
  - `audio` (file, required): Audio file to transcribe
  - `language` (string, optional): Language code (default: `en-US`)
  - `external_id` (string, optional): External reference ID

**Response:**
```json
{
  "success": true,
  "interaction_id": "int_abc123",
  "recording_id": "rec_xyz789",
  "transcript_id": "trans_def456",
  "status": "pending",
  "message": "Transcription in progress. Poll for results using the interaction ID."
}
```

### Get Transcript

**GET** `/api/transcribe/:interactionId`

Retrieve transcript for a specific interaction.

**Query Parameters:**
- `wait` (boolean, optional): If `true`, long-poll until completion

**Response:**
```json
{
  "success": true,
  "status": "completed",
  "is_complete": true,
  "transcript": {
    "transcript_id": "trans_def456",
    "status": "completed",
    "text": "Full transcript text here...",
    "segments": [
      {
        "id": "seg_001",
        "text": "Hello, how are you feeling today?",
        "start_time": 0.0,
        "end_time": 2.5,
        "speaker": "Doctor",
        "confidence": 0.95
      }
    ],
    "duration": 120.5,
    "language": "en-US"
  }
}
```

### Poll Transcript (Long-polling)

**GET** `/api/transcribe/:interactionId/poll`

Long-poll for transcript completion (60 second timeout).

**Response:** Same as Get Transcript

### Generate Clinical Document

**POST** `/api/transcribe/:interactionId/document`

Generate a clinical document from the transcript.

**Request:**
```json
{
  "template": "soap_note",
  "context": {
    "patient_name": "John Doe"
  }
}
```

**Response:**
```json
{
  "success": true,
  "status": "processing",
  "document_id": "doc_abc123"
}
```

---

## Ambient Streaming API

### Start Session

**POST** `/api/ambient/start`

Create a new ambient streaming session.

**Request:**
```json
{
  "language": "en-US",
  "external_id": "visit_123",
  "metadata": {
    "patient_id": "P001",
    "encounter_type": "follow-up"
  }
}
```

**Response:**
```json
{
  "success": true,
  "interaction_id": "int_abc123",
  "websocket_url": "ws://localhost:8080/api/ambient/ws/int_abc123",
  "status": "created",
  "message": "Ambient session created. Connect to the WebSocket URL to start streaming."
}
```

### Get Session Status

**GET** `/api/ambient/session/:interactionId`

Get current session status and accumulated data.

**Response:**
```json
{
  "success": true,
  "interaction_id": "int_abc123",
  "status": "active",
  "session_state": {
    "interaction_id": "int_abc123",
    "status": "active",
    "started_at": "2024-01-15T10:30:00Z",
    "transcripts": [
      {
        "type": "final",
        "text": "Patient reports chest pain for two days.",
        "speaker": "Patient",
        "is_final": true
      }
    ],
    "medical_events": [
      {
        "type": "fact",
        "category": "symptom",
        "value": "chest pain",
        "confidence": 0.92
      }
    ]
  }
}
```

### Get Server Stats

**GET** `/api/ambient/stats`

Get server-wide statistics.

**Response:**
```json
{
  "active_sessions": 3
}
```

---

## WebSocket Protocol

### Connection

**WS** `/api/ambient/ws/:interactionId`

Connect after creating a session via POST `/api/ambient/start`.

### Audio Streaming

Send raw binary PCM16 audio frames directly:
- Sample rate: 16000 Hz
- Channels: 1 (mono)
- Encoding: 16-bit signed integer (little-endian)
- Recommended frame size: 2048-4096 samples

### Control Messages

Send JSON messages for control:

```json
// Ping (keep-alive)
{ "type": "ping" }

// Stop session
{ "type": "stop" }

// Update configuration
{ "type": "config", "data": { "language": "es-ES" } }
```

### Server Messages

All server messages follow this format:
```json
{
  "type": "message_type",
  "data": { /* payload */ },
  "error": "error message if type is error",
  "timestamp": 1705312200000
}
```

**Message Types:**

| Type | Description | Data Fields |
|------|-------------|-------------|
| `connected` | Connection established | `interaction_id`, `status` |
| `transcript` | Transcript event | `text`, `is_final`, `speaker`, `type` |
| `medical_event` | Medical entity detected | `category`, `value`, `confidence` |
| `status` | Status update | Various |
| `pong` | Ping response | None |
| `error` | Error occurred | None (check `error` field) |
| `disconnected` | Connection closed | `reason` |

---

## Error Responses

All errors follow this format:
```json
{
  "success": false,
  "error": "Human-readable error message",
  "request_id": "uuid-for-tracking"
}
```

### HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Resource created |
| 202 | Request accepted (async processing) |
| 400 | Bad request (validation error) |
| 401 | Unauthorized |
| 404 | Resource not found |
| 413 | File too large |
| 415 | Unsupported media type |
| 500 | Internal server error |

---

## Rate Limits

For production deployments, consider implementing:
- API rate limiting per client
- WebSocket connection limits
- Audio frame rate limiting
- Concurrent session limits

