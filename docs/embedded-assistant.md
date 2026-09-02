# Embedded Assistant — Module Documentation

## What It Does

The Embedded Assistant integrates Corti's `<corti-embedded>` web component into the app as a full-page AI clinical assistant. Users can create clinical encounters, record conversations, and get AI-generated documentation — all inside the app without leaving to the Corti platform.

Access is gated by the `embedded_assistant.access` permission (RBAC). Roles that include it by default: **Superuser**, **Doctor**.

---

## Architecture Overview

```
Browser                    Our Backend               Corti
------                     -----------               -----
EmbeddedAssistantPage.vue
  └── useCortiEmbedded.ts
        └── GET /api/embedded/token  ──►  EmbeddedHandler
                                           └── EmbeddedTokenCache
                                                 └── POST CORTI_AUTH_URL (ROPC)  ──►  Keycloak
                                                       ◄── access_token
                                                           refresh_token
                                                           id_token
        ◄── { access_token, refresh_token, id_token }
  └── cortiApi.auth({ tokens })  ──────────────────────────────────►  <corti-embedded>
  └── cortiApi.createInteraction(...)
  └── cortiApi.startRecording() / stopRecording()
```

**Key design decision:** Corti credentials never reach the browser. The backend holds them, exchanges them for tokens via ROPC, caches the result, and only sends the short-lived tokens to the frontend. The frontend passes those directly to the web component's `auth()` call.

---

## Files

| File | Purpose |
|------|---------|
| `backend/internal/corti/embedded_auth.go` | `EmbeddedTokenCache` — ROPC token fetch, refresh, caching |
| `backend/internal/handlers/embedded_handler.go` | `GET /api/embedded/token` HTTP handler |
| `frontend/src/composables/useCortiEmbedded.ts` | All web component lifecycle logic (init, auth, events, API wrappers) |
| `frontend/src/views/EmbeddedAssistantPage.vue` | Page UI — header, status chip, encounter dialog, recording controls |

---

## Backend — `EmbeddedTokenCache` (`embedded_auth.go`)

### Authentication Flow: ROPC

Uses **Resource Owner Password Credentials (ROPC)** grant — confirmed by Corti support as the correct flow for backend-initiated auth on behalf of a shared demo user.

- Grant type: `password`
- Required scope: `openid profile email` — `openid` is **mandatory**, without it Keycloak does not return `id_token` and the web component auth call fails
- Endpoint: `CORTI_AUTH_URL` (same URL as all other Corti OAuth)

### Token Lifetimes

| Token | Lifetime |
|-------|---------|
| `access_token` | ~5 minutes (from Keycloak `expires_in`) |
| `refresh_token` | ~30 days (from Keycloak `refresh_expires_in`) |

### Caching & Refresh Strategy

```
GetToken() called
  ├── access_token valid (>60s remaining) → return cached (fast path, read lock)
  ├── access_token expiring, refresh_token valid → fetchViaRefresh() (slow path, write lock)
  └── both expired / first call → fetchViaROPC() (full exchange)
```

- 60-second proactive refresh buffer (same pattern as `TokenManager` for other Corti clients)
- Double-checked locking (read lock → check → write lock → re-check) to avoid stampede under concurrent requests
- A single shared token is used for all demo users — no per-user Corti account needed

### Handler (`embedded_handler.go`)

`GET /api/embedded/token` — protected by `authMW + RequirePermission(embedded_assistant.access)`

Returns:
```json
{
  "success": true,
  "access_token": "...",
  "refresh_token": "...",
  "id_token": "...",
  "token_type": "Bearer"
}
```

---

## Frontend — `useCortiEmbedded.ts`

A Vue 3 composable that owns the entire lifecycle of the `<corti-embedded>` web component.

### Initialization Sequence

```
initialize(el)
  1. Cast element to CortiEmbeddedAPI
  2. await 'embedded.ready' event (web component signals it's loaded)
  3. Wire event listeners: 'error', 'recording.started', 'recording.stopped'
  4. authenticate() — fetch tokens from backend, call cortiApi.auth()
  5. configure() — set features, appearance, locale
  6. cortiApi.navigate('/') — show the home screen
```

### Configuration Applied

```typescript
features: {
  navigation:        false,   // hide Corti's built-in nav (we control routing)
  documentFeedback:  false,
  aiChat:            true,
  templateEditor:    true,
  interactionTitle:  true,
  virtualMode:       true,
  syncDocumentAction: false,
}
appearance: { primaryColor: '#1565C0' }
locale: { interfaceLanguage: 'en', dictationLanguage: 'en' }
```

### Error Recovery

If the web component fires an `UNAUTHORIZED` error code, `handleError` silently re-fetches tokens and re-authenticates without showing an error to the user. Any other error surfaces as an alert banner.

### Public Methods Exposed

| Method | What it does |
|--------|-------------|
| `initialize(el)` | Full init sequence (called in `onMounted`) |
| `createInteraction(type, title?)` | Creates a clinical encounter, navigates web component to the session |
| `startRecording()` | Delegates to `cortiApi.startRecording()` |
| `stopRecording()` | Delegates to `cortiApi.stopRecording()` |
| `getStatus()` | Returns current web component status |

### Encounter Types Supported

`first_consultation`, `ambulatory`, `virtual`, `emergency`, `inpatient_encounter`, `home_health`

---

## Page UI — `EmbeddedAssistantPage.vue`

- Header with connection status chip (Loading → Authenticating → Connected)
- **New Encounter** button (shown when authenticated, no active interaction) → dialog to pick encounter type and optional title
- **Start/Stop Recording** buttons (shown contextually based on interaction and recording state)
- The `<corti-embedded>` web component fills the card below, height = `calc(100vh - 220px)`
- Error alert banner for any failures

---

## Environment Variables Required

Add these to `backend/.env`:

```env
CORTI_EMBEDDED_CLIENT_ID=<client id from Corti Admin API>
CORTI_EMBEDDED_USERNAME=<shared backend user email>
CORTI_EMBEDDED_PASSWORD=<shared backend user password>
```

- The shared user must exist in the Corti platform (created via the Corti Admin API, not the regular UI)
- `CORTI_AUTH_URL` is shared with the main token manager — no extra URL needed
- If any of the three vars are missing, the backend logs a warning at startup and the `/api/embedded/token` endpoint returns 500

---

## RBAC Integration

- Route `/embedded-assistant` requires `embedded_assistant.access` permission (enforced in `router/index.ts`)
- API endpoint `GET /api/embedded/token` requires `embedded_assistant.access` (enforced by `RequirePermission` middleware in `main.go`)
- Menu item is hidden via `App.vue` nav filter when the permission is absent
- Permission can be granted/denied per-user via the User Management → Permission Overrides dialog

---

## Known Constraints

- **Single shared Corti user** — all app users share one Corti account token. Interactions created by different app users appear under the same Corti identity. Acceptable for demo; for production, each user would need their own Corti account.
- **No token invalidation on logout** — the cached token stays alive on the backend until it naturally expires. This is fine for a shared demo token.
- **Web component is an external package** — loaded from `@corti/embedded-web` npm package. If Corti releases a breaking API change, `useCortiEmbedded.ts` is the only file that needs updating.
- **`baseurl`** is hardcoded to `https://assistant.us.corti.app` in `useCortiEmbedded.ts`. If the deployment region changes, update this constant.
