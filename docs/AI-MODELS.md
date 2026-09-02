# AI Model Dependencies

Every AI model and AI service this application depends on, what it is used for,
how it is provisioned, and what breaks without it.

Installation commands live in [SETUP.md § 5](SETUP.md#5-step-2--ai-models);
this document is the dependency reference behind them.

---

## 1. Summary

| # | Dependency | Used for | Where it runs | Required? |
|---|-----------|----------|---------------|-----------|
| 1 | **Corti API** (cloud) | Speech-to-text, clinical document generation | Corti cloud (`api.us.corti.app`) | Only for recording and file transcription |
| 2 | **DeepSeek v4 Flash** (`deepseek-v4-flash`) | AI Assistance: 15-field clinical extraction + grounded Q&A | Hosted provider for the portable live-demo profile | Only for the AI Assistance panel |
| 3 | **bge-small-en-v1.5** | Embeddings for semantic search | Local Ollama (or any OpenAI-compatible embeddings endpoint) | Only for semantic search |
| 4 | **`@corti/embedded-web`** | Corti's own embedded assistant widget | Browser (npm package + Corti cloud) | Only for the `/embedded-assistant` page |

None of the four blocks server startup. Missing an LLM or embedder degrades the
matching feature and is reported by `GET /api/ai/status`; missing PostgreSQL, by
contrast, is fatal.

Models 2 and 3 are reached over the **same OpenAI-compatible protocol** but are
configured independently (`AI_*` vs `EMBED_*`), because a hosted chat provider
usually serves no embedding models.

### Current live-demo profile

The checked-in `backend/.env.example` targets the provider's low-cost DeepSeek v4
Flash model so the demo can run on an instructor's machine without a local GPU
or Ollama installation. The provider guide is linked from that file and the
regional endpoints are documented in [SETUP.md § 6](SETUP.md#6-step-3--backend-configuration).

```env
AI_BASE_URL=https://api.llm-token.cn/v1
AI_MODEL=deepseek-v4-flash
AI_API_KEY=<set locally; never commit>
AI_TEMPERATURE=0.2
AI_TIMEOUT_SECONDS=300
AI_MAX_COMPLETION_TOKENS=4096
```

For instructors outside mainland China, use the nearest regional Base URL from
the setup guide; the model name and key stay the same. The provider's `/v1/models`
catalog and a minimal `/v1/chat/completions` request were verified for this
configuration on 2026-08-12.

---

## 2. Corti API — speech-to-text and clinical documents

| | |
|---|---|
| **Vendor model** | Corti's proprietary ASR and clinical documentation models |
| **Access** | OAuth2 client credentials; the Go backend proxies every call so credentials never reach the browser |
| **Endpoints** | `https://auth.us.corti.app/...` (token), `https://api.us.corti.app` (REST), `wss://api.us.corti.app/stt/stream` (streaming) |
| **Env vars** | `CORTI_CLIENT_ID`, `CORTI_CLIENT_SECRET`, `CORTI_TENANT`, `CORTI_ENVIRONMENT`, `CORTI_AUTH_URL`, `CORTI_API_BASE_URL`, `CORTI_WS_URL` |
| **Credentials from** | <https://console.corti.app> |
| **Code** | `internal/corti/` — `token_manager.go`, `async_client.go`, `ambient_proxy.go`, `dictation_proxy.go` |

Used by three features: file transcription (async upload), ambient streaming
(live WebSocket), and dictation. Tokens are cached server-side and refreshed
60 seconds before expiry (`TOKEN_REFRESH_BUFFER`).

Without credentials: login, saved sessions, semantic search, and the AI
Assistance module all still work — on transcripts that already exist or are
pasted in. Only new transcription fails.

Audio format for streaming: PCM16, 16 kHz, mono.

---

## 3. Qwythos-9B — the AI Assistance chat model

| | |
|---|---|
| **Model** | Qwythos-9B (Claude-Mythos-5 distill), Q4_K_M GGUF, ~6.8 GB on disk |
| **Ollama tag** | `qwythos-16k-chat` — **built from `deploy/ai-vm/Modelfile`, never pulled directly** |
| **Base** | `hf.co/empero-ai/Qwythos-9B-Claude-Mythos-5-1M-GGUF:Q4_K_M` |
| **Context** | 16,384 tokens (Modelfile `num_ctx`; the base model supports far more) |
| **Sampling** | temperature 0.6, top_p 0.95 — greedy sampling loops on this model |
| **Protocol** | OpenAI `/v1/chat/completions` |
| **Env vars** | `AI_BASE_URL`, `AI_MODEL`, `AI_API_KEY`, `AI_TEMPERATURE`, `AI_TIMEOUT_SECONDS`, `AI_MAX_COMPLETION_TOKENS` |
| **Code** | `internal/ai/` (`client.go`, `openai_compat.go`, `prompts.go`, `types.go`), `handlers/ai_handler.go` |
| **Endpoints served** | `POST /api/ai/summarize`, `POST /api/ai/chat`, `GET /api/ai/status` — all gated by the `ambient.access` permission |

### Build it correctly

```bash
ollama create qwythos-16k-chat -f deploy/ai-vm/Modelfile
```

> **The single most damaging misconfiguration in this project.** Pulling the
> GGUF directly (`ollama pull hf.co/empero-ai/Qwythos-9B-...`) makes Ollama fail
> to convert the embedded Jinja chat template and fall back to
> `TEMPLATE {{ .Prompt }}`, which **silently drops system messages**. Since the
> transcript and every instruction live in the system message, extraction and
> grounding then fail while looking like a model-quality problem. The old
> `qwythos-16k` tag has the same defect. Any new host — including a freshly
> provisioned VM — must run the Modelfile build.

### What it produces

`POST /api/ai/summarize` returns a nested 15-field `extraction` object
(camelCase): `summary`, `chiefComplaint`, `history`, `allergies`, `medications`,
`symptoms`, `diagnosis`, `differentialDiagnosis`, `treatment`, `labTests`,
`procedures`, `followUp`, `riskFactors`, `medicalTerms`, `actionItems`.

`POST /api/ai/chat` grounds answers in the stored extraction when the client
sends one (far cheaper in prompt tokens) and falls back to the raw transcript.
Both the transcript and the extraction are injected inside delimiters
(`<transcript>`, `<consultation_record>`) in the system message, so a user
question can never be mixed into the grounding material.

### Server-side handling of model quirks

`internal/ai/openai_compat.go` compensates for known fine-tune artifacts:

- `<think>…</think>` reasoning traces are stripped before the answer is returned.
- Identity boilerplate ("I am Qwythos created by…", a leading `Qwythos:` prefix)
  is removed.
- If sanitizing leaves an empty answer, the request is retried once; a second
  empty result returns `ErrRunawayReasoning`.

Keep `AI_TEMPERATURE` at 0.6 or above — the retry relies on getting a different
sample.

### Sizing and latency

| Hardware | Fits? | Extraction latency |
|----------|-------|--------------------|
| NVIDIA L4 / A10G / RTX 4090 (24 GB) | comfortably | seconds — recommended |
| T4 (16 GB) | yes | slower prompt processing, still far faster than CPU |
| 12 GB VRAM (RTX 3060) | yes | model ~6.5 GB + 16k KV cache fits |
| Apple Silicon / CPU, 16 GB RAM | yes | ~2–2.5 min per extraction (~14 tok/s) — development only |

Weights ~5.6 GB; budget ~10 GB VRAM with the 16k context.

`AI_MAX_COMPLETION_TOKENS=4096` is a measured value, not a guess: the budget has
to cover the hidden reasoning trace *plus* the dense 15-field extraction. 2048
truncated intermittently on the same transcript; 1024 always truncated.

---

## 4. bge-small-en-v1.5 — the embedding model

| | |
|---|---|
| **Model** | `hf.co/CompendiumLabs/bge-small-en-v1.5-gguf`, ~24 MB |
| **Dimensions** | **384** — hard-coded as `ai.EmbeddingDimensions` and in every `vector(384)` column |
| **Protocol** | OpenAI `/v1/embeddings` |
| **Env vars** | `EMBED_BASE_URL`, `EMBED_MODEL`, `EMBED_API_KEY`, `EMBED_TIMEOUT_SECONDS` |
| **Code** | `internal/ai/embed.go` (client + chunker), `internal/embedding/worker.go` |
| **Endpoint served** | `POST /api/search` |

```bash
ollama pull hf.co/CompendiumLabs/bge-small-en-v1.5-gguf
```

Details that matter:

- **384, not 368.** The original task list specified "bge-1.5 368 dimensions",
  which does not exist — the v1.5 family is small=384, base=768, large=1024.
  See [adr/0002](adr/0002-bge-small-384-not-368.md).
- Queries are embedded with the bge retrieval prefix
  (`"Represent this sentence for searching relevant passages: "`); passages are
  embedded without it. Mixing this up quietly degrades ranking.
- Transcripts are chunked at ~280 words with a 50-word overlap, staying inside
  the model's 512-token window.
- One embedding call per session covers all chunks, the whole transcript, and
  the extraction serialization.

The dimension is a schema-level commitment: swapping to a different embedding
model means altering every vector column and re-embedding all stored content
(see [DATABASE.md § 4](DATABASE.md#4-embedding-lifecycle)).

---

## 5. `@corti/embedded-web` — the embedded assistant widget

A frontend npm dependency (`^0.3.1`) that renders Corti's own assistant on
`/embedded-assistant`. It authenticates with a short-lived token minted by
`GET /api/embedded/token`, which uses ROPC credentials
(`CORTI_EMBEDDED_CLIENT_ID`, `CORTI_EMBEDDED_USERNAME`, `CORTI_EMBEDDED_PASSWORD`)
against a shared backend user provisioned through the Corti Admin API.

Without those three variables that endpoint returns 500 and only that page is
affected. Full description in [embedded-assistant.md](embedded-assistant.md).

---

## 6. Deployment options for the chat model

The backend is provider-agnostic — three variables select the deployment.

### A. Local Ollama (development)

```env
AI_BASE_URL=http://localhost:11434/v1
AI_MODEL=qwythos-16k-chat
AI_API_KEY=
AI_TIMEOUT_SECONDS=300
```

Free, private, slow (~2–2.5 min per extraction on a laptop). Ollama must be
running (`ollama serve`, `brew services start ollama`, or the Ollama.app).

### B. Private GPU VM

```env
AI_BASE_URL=http://localhost:8443/v1     # local end of the SSH tunnel
AI_MODEL=qwythos-16k-chat
AI_API_KEY=<token generated by setup.sh>
```

```
Go backend ──► localhost:8443 (SSH tunnel) ──► nginx :8443 (Bearer auth) ──► Ollama :11434 (localhost only)
```

One-time provisioning, then daily start/stop:

```bash
bash deploy/ai-vm/provision_gcp.sh    # g2-standard-4, 1x L4 24GB, ~$0.70–0.85/hr
bash deploy/ai-vm/manage.sh start     # start VM + tunnel, health-check it
bash deploy/ai-vm/manage.sh status
bash deploy/ai-vm/manage.sh stop      # stops GPU billing (disk still costs)
```

Notes:

- **Ollama has no authentication of its own.** `setup.sh` binds it to localhost;
  only nginx, which checks `Authorization: Bearer`, listens externally.
- The nginx proxy is plain HTTP, so it must never carry transcripts across the
  public internet unencrypted — use the SSH tunnel (default), a VPC, WireGuard/
  Tailscale, or put TLS in front. The controller script enforces this by
  refusing any `AI_BASE_URL` that is neither the local tunnel nor `https://`.
- GPU zones frequently run out of L4 capacity; both scripts fail over across a
  zone list and roll back partial resources.
- `OLLAMA_KEEP_ALIVE=-1` keeps the model resident (no cold start);
  `OLLAMA_NUM_PARALLEL=2` allows one summarize and one chat concurrently.

Full details: [deploy/ai-vm/README.md](../deploy/ai-vm/README.md).

### C. Hosted OpenAI-compatible provider

```env
AI_BASE_URL=https://api.llm-token.cn/v1
AI_MODEL=deepseek-v4-flash
AI_API_KEY=<provider key>
AI_TEMPERATURE=0.2
```

This is the live-demo profile: no GPU or local model download is needed, and the
selected DeepSeek v4 Flash model is cheaper than the provider's v4 Pro model.
Transcripts leave the instructor's machine, so use only approved demo or
synthetic clinical data and confirm the provider is acceptable for the intended
data-handling agreement.

A convention already in use in `backend/.env`: when switching providers, the
previous configuration is kept alongside as `ROLLBACK_AI_BASE_URL`,
`ROLLBACK_AI_MODEL`, `ROLLBACK_AI_API_KEY`, `ROLLBACK_AI_TEMPERATURE`. These are
**not read by the application** — they are a note-to-self for reverting. Rename
them back over the `AI_*` keys to roll back.

---

## 7. Swapping models

| Change | Procedure |
|--------|-----------|
| Different chat model, same host | `ollama pull llama3.1:8b`, set `AI_MODEL=llama3.1:8b`, restart the backend |
| Chat model → hosted provider | Set the three `AI_*` variables; nothing to install |
| Fine-tuned chat model | Convert to GGUF → `ollama create <tag> -f <Modelfile>` → set `AI_MODEL` |
| Larger context | Edit `PARAMETER num_ctx` in `deploy/ai-vm/Modelfile`, rebuild the tag |
| **Different embedding model** | Only with the same 384 dimensions. Otherwise: change `ai.EmbeddingDimensions`, alter every `vector(384)` column, and re-embed everything (`UPDATE sessions SET embedded_at = NULL;`) |

The extraction prompt is model-agnostic. A different model may need
`AI_MAX_COMPLETION_TOKENS` and `AI_TEMPERATURE` revisited; the Qwythos-specific
artifact stripping in `openai_compat.go` is harmless for DeepSeek.

---

## 8. What leaves the machine

| Data | Sent to | When |
|------|---------|------|
| Audio | Corti cloud | Recording or file transcription |
| Transcript | Corti cloud | Clinical document generation |
| Transcript | Chat model endpoint | AI extraction (summarize) |
| Extraction, or transcript | Chat model endpoint | AI chat question |
| Transcript chunks + extraction text | Embedding endpoint | Asynchronously after every session save |
| Search query text | Embedding endpoint | Every semantic search |

With Ollama on localhost, categories 2–6 never leave the machine. With the GPU
VM they cross an SSH tunnel to your own instance. With a hosted provider they
reach a third party — the deciding factor when choosing option C.

---

## 9. Verifying the model layer

```bash
# Ollama is up and lists both models
ollama list | grep -E 'qwythos-16k-chat|bge-small'

# Embedding dimension — must print 384
curl -s http://localhost:11434/v1/embeddings -H 'Content-Type: application/json' \
  -d '{"model":"hf.co/CompendiumLabs/bge-small-en-v1.5-gguf","input":["chest pain"]}' \
  | jq '.data[0].embedding | length'

# Backend's own view (needs a token — see SETUP.md § 9)
curl -s http://localhost:8090/api/ai/status -H "Authorization: Bearer $JWT" | jq
# → {"success":true,"status":{"reachable":true,"model":"...","model_available":true,"base_url":"..."}}
```

`GET /api/ai/status` calls the endpoint's `/models` list: `reachable` means the
HTTP call succeeded, `model_available` means `AI_MODEL` appeared in the response
(Ollama tags such as `name:latest` are matched with and without the tag).

---

## 10. Failure modes

| Symptom | Cause | Fix |
|---------|-------|-----|
| `reachable:false` | Endpoint down or wrong URL | Start Ollama / the VM tunnel; `curl $AI_BASE_URL/models` |
| `model_available:false` | `AI_MODEL` not served there | `ollama list` and align the value |
| Answers ignore the transcript; extraction is garbage | Broken `{{ .Prompt }}` template dropping system messages | Rebuild from `deploy/ai-vm/Modelfile`; never use the raw pull or `qwythos-16k` |
| Truncated extraction JSON | `AI_MAX_COMPLETION_TOKENS` too low | 4096 or higher |
| Timeouts | CPU inference | Raise `AI_TIMEOUT_SECONDS`, or use the GPU VM |
| "I am Qwythos…" / reasoning-only replies | Fine-tune artifacts | Handled by `openai_compat.go`; keep temperature ≥ 0.6 |
| Repetitive looping output | Temperature too low | Keep ≥ 0.6 |
| `expected 384 dimensions, got N` | Wrong embedding model | Only bge-small-en-v1.5 matches the schema |
| Search empty on a populated database | Embeddings pending or embedder down | See [DATABASE.md § 4](DATABASE.md#4-embedding-lifecycle) |
| `ZONE_RESOURCE_POOL_EXHAUSTED` | No L4 capacity in that zone | Scripts fail over automatically; or set `ZONES="..."` |
| VM answers slowly on the first request after a restart | Model not resident yet | `setup.sh` sends a warm-up request; check `OLLAMA_KEEP_ALIVE=-1` |
