#!/usr/bin/env bash
# AI Assistance model server — one-shot VM setup.
#
# Run ON the VM (Ubuntu 22.04/24.04, NVIDIA driver present for GPU boxes):
#   sudo bash setup.sh
#
# What it does:
#   1. Installs Ollama and tunes it for server use (model kept loaded,
#      flash attention, localhost-only bind).
#   2. Pulls the Qwythos GGUF and builds the chat-ready 16k model from the
#      Modelfile next to this script.
#   3. Installs nginx as a bearer-token auth proxy on :8443 (Ollama itself
#      has NO auth — it must never be exposed directly).
#   4. Prints the AI_BASE_URL / AI_API_KEY values for the backend .env.
#
# SECURITY: the proxy speaks plain HTTP. Clinical transcripts must not cross
# the public internet unencrypted — reach this VM over a private VPC network,
# WireGuard/Tailscale, or an SSH tunnel, or put TLS in front (see README).
set -euo pipefail

PROXY_PORT="${PROXY_PORT:-8443}"
MODEL_NAME="qwythos-16k-chat"
BASE_MODEL="hf.co/empero-ai/Qwythos-9B-Claude-Mythos-5-1M-GGUF:Q4_K_M"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ $EUID -ne 0 ]]; then
  echo "Run with sudo: sudo bash $0" >&2
  exit 1
fi
if [[ ! -f "$SCRIPT_DIR/Modelfile" ]]; then
  echo "Modelfile not found next to setup.sh — copy the whole deploy/ai-vm/ directory to the VM." >&2
  exit 1
fi

echo "==> [1/4] Installing Ollama"
command -v ollama >/dev/null || curl -fsSL https://ollama.com/install.sh | sh

mkdir -p /etc/systemd/system/ollama.service.d
cat > /etc/systemd/system/ollama.service.d/override.conf <<'EOF'
[Service]
# Localhost only — nginx is the sole way in.
Environment="OLLAMA_HOST=127.0.0.1:11434"
# Keep the model resident so requests never pay the load penalty.
Environment="OLLAMA_KEEP_ALIVE=-1"
Environment="OLLAMA_FLASH_ATTENTION=1"
# Two concurrent requests covers summarize + a chat question.
Environment="OLLAMA_NUM_PARALLEL=2"
EOF
systemctl daemon-reload
systemctl enable --now ollama
systemctl restart ollama
sleep 3

echo "==> [2/4] Pulling base model (~5.6 GB) and building $MODEL_NAME"
ollama pull "$BASE_MODEL"
(cd "$SCRIPT_DIR" && ollama create "$MODEL_NAME" -f Modelfile)

echo "==> [3/4] Installing nginx bearer-token proxy on :$PROXY_PORT"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq && apt-get install -y -qq nginx openssl

TOKEN_FILE=/etc/ai-proxy-token
if [[ ! -s "$TOKEN_FILE" ]]; then
  openssl rand -hex 32 > "$TOKEN_FILE"
  chmod 600 "$TOKEN_FILE"
fi
TOKEN="$(cat "$TOKEN_FILE")"

sed -e "s|__TOKEN__|$TOKEN|g" -e "s|__PORT__|$PROXY_PORT|g" \
  "$SCRIPT_DIR/nginx-ai-proxy.conf" > /etc/nginx/sites-available/ai-proxy
ln -sf /etc/nginx/sites-available/ai-proxy /etc/nginx/sites-enabled/ai-proxy
rm -f /etc/nginx/sites-enabled/default
nginx -t
systemctl enable --now nginx
systemctl reload nginx

echo "==> [4/4] Warming the model (first token after this is fast)"
curl -s http://127.0.0.1:11434/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d "{\"model\":\"$MODEL_NAME\",\"messages\":[{\"role\":\"user\",\"content\":\"Say ready.\"}],\"max_tokens\":16}" >/dev/null || true

IP="$(hostname -I | awk '{print $1}')"
cat <<EOF

============================================================
 AI model server ready.

 Backend .env settings (medical-transcription/backend/.env):

   AI_BASE_URL=http://$IP:$PROXY_PORT/v1
   AI_MODEL=$MODEL_NAME
   AI_API_KEY=$TOKEN

 Smoke test from the backend host:

   curl http://$IP:$PROXY_PORT/v1/models -H "Authorization: Bearer $TOKEN"

 Reminder: use a private network / VPN / SSH tunnel or add TLS —
 do not send transcripts to this address over the public internet.
============================================================
EOF
