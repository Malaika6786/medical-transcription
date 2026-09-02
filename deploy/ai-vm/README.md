# AI Assistance — model server deployment

Serves the `qwythos-16k-chat` model (Qwythos-9B Q4_K_M GGUF via Ollama) behind
a bearer-token nginx proxy, for the backend's AI Assistance module
(`backend/internal/ai/`). The backend talks plain OpenAI protocol, so this VM
is swappable with local Ollama or any hosted provider by changing two env vars.

```
Go backend ── AI_BASE_URL/v1/chat/completions (Bearer AI_API_KEY) ──► nginx :8443 ──► Ollama :11434 (localhost only)
```

## Files

| File | Purpose |
|------|---------|
| `Modelfile` | Chat-ready 16k-context build of the Qwythos GGUF (fixes the broken template Ollama imports by default — see comments inside) |
| `setup.sh` | One-shot setup, run **on the VM** with sudo |
| `nginx-ai-proxy.conf` | Proxy template used by `setup.sh` |
| `provision_gcp.sh` | Optional: create a GCP `g2-standard-4` (L4 24GB) VM and run `setup.sh` on it, run **from your laptop** |
| `docker-compose.yml` + `nginx-docker.conf.template` | Docker alternative to `setup.sh` |

## Quickstart (any Ubuntu VM)

```bash
# from the repo root, on your laptop:
scp -r deploy/ai-vm user@VM_IP:~/
ssh user@VM_IP sudo bash ai-vm/setup.sh
```

The script prints the three values to put in `backend/.env`:

```
AI_BASE_URL=http://VM_IP:8443/v1
AI_MODEL=qwythos-16k-chat
AI_API_KEY=<generated token>
AI_MAX_COMPLETION_TOKENS=4096
```

Restart the Go backend; `GET /api/ai/status` should report
`reachable: true, model_available: true`.

## Quickstart (GCP, one command)

```bash
bash deploy/ai-vm/provision_gcp.sh
```

Costs ~$0.70–0.85/hr while running — stop the instance when idle (commands
printed by the script). Requires GPU quota ≥ 1.

## Everyday start and stop

After the one-time provisioning and backend `.env` configuration, use the
management helper from the repository root:

```bash
# Start the VM, open the private localhost:8443 SSH tunnel, and verify it:
bash deploy/ai-vm/manage.sh start

# Check the VM and tunnel:
bash deploy/ai-vm/manage.sh status

# Close the tunnel and stop GPU billing:
bash deploy/ai-vm/manage.sh stop
```

Keep `AI_BASE_URL=http://localhost:8443/v1` in `backend/.env`. The helper reads
the existing `AI_API_KEY` from that file for its health check without printing
the token.

## Zone failover

GPU zones frequently run out of L4 capacity (`ZONE_RESOURCE_POOL_EXHAUSTED`).
Both scripts retry across a zone list automatically:

- `provision_gcp.sh` tries each zone in `ZONES` in order; after a capacity
  failure it deletes the partially created instance (rollback) before moving on.
- `manage.sh start` discovers the VM's current zone itself; if the start fails
  for capacity reasons it creates a temporary machine image and restores a
  replacement in the next zone. The stopped source VM is deleted only after the
  replacement is confirmed `RUNNING` and its authenticated model endpoint passes
  a health check. If replacement creation, verification, or source deletion
  fails, the owned replacement is removed and the original VM remains stopped.
  Temporary machine images are deleted automatically; a cleanup warning includes
  the image name if manual deletion is needed.

Default: `ZONES="us-central1-b us-central1-a us-central1-c us-central1-f"`.
Override with e.g. `ZONES="us-central1-a us-central1-c" bash deploy/ai-vm/manage.sh start`.
Keep `ZONES` in one region so the cloned network configuration remains valid.
Non-capacity errors (quota, permissions) abort immediately — retrying zones
won't fix those.

## VM sizing

| Hardware | Fits? | Rough experience |
|----------|-------|------------------|
| NVIDIA L4 / A10G / RTX 4090 (24 GB) | comfortable | summarize in a few seconds; recommended |
| T4 (16 GB) | yes | slower prompt processing, still far faster than CPU |
| 12 GB VRAM (RTX 3060 etc.) | yes | model ~6.5 GB + 16k KV cache fits |
| CPU-only (8+ cores, 16 GB RAM) | works | tens of seconds per response — dev fallback only |

The Q4_K_M weights are ~5.6 GB; budget ~10 GB VRAM with the 16k context.

## Security — read before exposing anything

- **Ollama itself has no authentication.** `setup.sh` binds it to localhost;
  only nginx (which checks `Authorization: Bearer <token>`) listens externally.
- **The proxy is plain HTTP.** Clinical transcripts must not cross the public
  internet unencrypted. Use one of:
  - same private network / VPC as the backend (best),
  - Tailscale or WireGuard between backend and VM,
  - an SSH tunnel (`ssh -N -L 8443:localhost:8443 user@VM_IP`, then
    `AI_BASE_URL=http://localhost:8443/v1`),
  - or add TLS (e.g. Caddy with a domain) in front of nginx.
- Don't open :8443 in the VM firewall unless one of the above is in place.
- Rotate the token by deleting `/etc/ai-proxy-token` and re-running `setup.sh`.

## Performance notes

- `OLLAMA_KEEP_ALIVE=-1` keeps the model in memory permanently — no cold-start
  latency per request (that's most of the "fast generation" win besides the GPU).
- `setup.sh` fires one warm-up request so even the first user request is fast.
- `OLLAMA_NUM_PARALLEL=2` allows a summarize and a chat question concurrently;
  raise it if multiple clinicians will use the panel at once (costs VRAM).
- Raise the context by editing `PARAMETER num_ctx` in the Modelfile and
  re-running `ollama create qwythos-16k-chat -f Modelfile` (the base model
  supports up to 1M tokens; 16k covers single consultations with headroom).

## Swapping models later

The backend is provider-agnostic. On the VM:

```bash
ollama pull llama3.1:8b        # or any other model
```

then set `AI_MODEL=llama3.1:8b` in `backend/.env`. A future fine-tuned model
is deployed the same way: convert to GGUF, `ollama create`, update `AI_MODEL`.
