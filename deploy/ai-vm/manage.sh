#!/usr/bin/env bash
# Start/stop the AI model VM and its private local SSH tunnel.
set -euo pipefail

INSTANCE="${INSTANCE:-ai-model-server}"
ZONE="${ZONE:-}"            # auto-discovered from the instance when empty
LOCAL_PORT="${LOCAL_PORT:-8443}"
REMOTE_PORT="${REMOTE_PORT:-8443}"
# Fallback zones, in order, when the current zone is out of GPU capacity on
# start. Relocation clones the stopped VM from a temporary machine image, then
# deletes the source only after the replacement is confirmed RUNNING.
ZONES="${ZONES:-us-central1-b us-central1-a us-central1-c us-central1-f}"
RELOCATION_LABEL_KEY="medical_transcription_relocation"
RELOCATION_VERIFY_ATTEMPTS="${RELOCATION_VERIFY_ATTEMPTS:-30}"
RELOCATION_VERIFY_DELAY="${RELOCATION_VERIFY_DELAY:-5}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ENV_FILE="$REPO_ROOT/backend/.env"

is_capacity_error() {
  grep -qiE 'ZONE_RESOURCE_POOL_EXHAUSTED|does not have enough resources available|stockout|out of capacity' <<<"$1"
}

resolve_zone() {
  local discovered count
  if [[ -z "$ZONE" ]]; then
    discovered="$(gcloud compute instances list --filter="name=$INSTANCE" \
      --format='value(zone)' 2>/dev/null || true)"
    count="$(awk 'NF { count++ } END { print count + 0 }' <<<"$discovered")"
    if [[ "$count" -gt 1 ]]; then
      echo "Multiple VMs named $INSTANCE were found; set ZONE explicitly:" >&2
      awk 'NF { print "  " $0 }' <<<"$discovered" >&2
      return 1
    fi
    ZONE="$(awk 'NF { print; exit }' <<<"$discovered")"
  fi
}

delete_machine_image() {
  local image="$1"
  if ! gcloud compute machine-images delete "$image" --quiet >/dev/null 2>&1; then
    echo "WARNING: could not delete temporary machine image $image; delete it manually to avoid storage charges." >&2
    return 1
  fi
}

replacement_probe() {
  local zone="$1" token="$2"
  local output count actual

  if ! output="$(gcloud compute instances list \
      --zones="$zone" \
      --filter="name=('$INSTANCE')" \
      --format="csv[no-heading](status,labels.$RELOCATION_LABEL_KEY)" 2>/dev/null)"; then
    return 2
  fi
  count="$(awk 'NF { count++ } END { print count + 0 }' <<<"$output")"
  [[ "$count" -eq 0 ]] && return 1
  [[ "$count" -eq 1 ]] || return 2
  actual="${output#*,}"
  [[ "$actual" == "$token" ]] || return 2
  return 0
}

delete_replacement() {
  local zone="$1" token="$2"
  local output probe_result

  if replacement_probe "$zone" "$token"; then
    :
  else
    probe_result=$?
    [[ "$probe_result" -eq 1 ]] && return 0
    echo "ERROR: refusing to delete $INSTANCE in $zone because relocation ownership could not be proven." >&2
    echo "WARNING: the destination VM may still be RUNNING; inspect it manually to stop possible GPU billing." >&2
    return 1
  fi

  gcloud compute instances stop "$INSTANCE" --zone="$zone" --quiet >/dev/null 2>&1 || true
  if ! output="$(gcloud compute instances delete "$INSTANCE" \
      --zone="$zone" --delete-disks=all --quiet 2>&1)"; then
    echo "$output" >&2
    echo "ERROR: could not roll back replacement VM $INSTANCE in $zone. It was stopped, but may need manual cleanup." >&2
    return 1
  fi
}

verify_remote_candidate() {
  local zone="$1"
  local token attempt

  token="$(sed -n 's/^AI_API_KEY=//p' "$ENV_FILE" 2>/dev/null || true)"
  if [[ -z "$token" ]]; then
    echo "AI_API_KEY is missing from $ENV_FILE; refusing to cut over an unverified replacement." >&2
    return 1
  fi

  for attempt in $(seq 1 "$RELOCATION_VERIFY_ATTEMPTS"); do
    if printf '%s\n' "$token" | gcloud compute ssh "$INSTANCE" \
        --zone="$zone" --quiet \
        --command="IFS= read -r token; curl -fsS --max-time 15 http://127.0.0.1:${REMOTE_PORT}/v1/models -H \"Authorization: Bearer \$token\" >/dev/null" \
        >/dev/null 2>&1; then
      return 0
    fi
    if [[ "$attempt" -lt "$RELOCATION_VERIFY_ATTEMPTS" ]]; then
      sleep "$RELOCATION_VERIFY_DELAY"
    fi
  done
  return 1
}

relocate_vm() {
  local source_zone="$1"
  local image token candidate output status probe_result
  local candidate_created=0

  image="relocate-${INSTANCE:0:28}-$(date +%Y%m%d-%H%M%S)-$$"
  token="r$(date +%Y%m%d%H%M%S)-$$-$RANDOM"
  echo "==> Capturing stopped $INSTANCE as temporary machine image $image"
  if ! output="$(gcloud compute machine-images create "$image" \
      --source-instance="$INSTANCE" \
      --source-instance-zone="$source_zone" 2>&1)"; then
    echo "$output" >&2
    echo "ERROR: could not create the temporary machine image; the source VM is unchanged." >&2
    return 1
  fi

  for candidate in $ZONES; do
    [[ "$candidate" == "$source_zone" ]] && continue
    if [[ "${candidate%-*}" != "${source_zone%-*}" ]]; then
      echo "ERROR: destination $candidate is outside the source region ${source_zone%-*}; refusing an unsafe network migration." >&2
      delete_machine_image "$image" || true
      return 1
    fi

    if ! output="$(gcloud compute instances list \
        --zones="$candidate" \
        --filter="name=('$INSTANCE')" \
        --format='value(name)' 2>/dev/null)"; then
      echo "ERROR: could not determine whether $candidate is safe for relocation." >&2
      echo "WARNING: retaining temporary machine image $image for recovery." >&2
      return 1
    fi
    if [[ -n "$output" ]]; then
      echo "ERROR: a VM named $INSTANCE already exists in $candidate; refusing to overwrite it." >&2
      delete_machine_image "$image" || true
      return 1
    fi

    echo "==> Creating replacement $INSTANCE in $candidate"
    candidate_created=0
    if output="$(gcloud compute instances create "$INSTANCE" \
        --zone="$candidate" \
        --source-machine-image="$image" \
        --labels="$RELOCATION_LABEL_KEY=$token" 2>&1)"; then
      candidate_created=1
    else
      echo "$output" >&2
      if replacement_probe "$candidate" "$token"; then
        candidate_created=1
      else
        probe_result=$?
        if [[ "$probe_result" -eq 2 ]]; then
          echo "ERROR: replacement creation was ambiguous; refusing unproven cleanup in $candidate." >&2
          echo "WARNING: retaining temporary machine image $image for recovery." >&2
          return 1
        fi
      fi
      if [[ "$candidate_created" == "1" ]] && ! delete_replacement "$candidate" "$token"; then
        echo "WARNING: retaining temporary machine image $image for recovery." >&2
        return 1
      fi
      if is_capacity_error "$output"; then
        echo "==> $candidate is also out of capacity; trying next zone"
        continue
      fi
      delete_machine_image "$image" || true
      return 1
    fi

    status="$(gcloud compute instances describe "$INSTANCE" \
      --zone="$candidate" --format='value(status)' 2>/dev/null || true)"
    if [[ "$status" != "RUNNING" ]] || ! replacement_probe "$candidate" "$token"; then
      echo "ERROR: replacement VM in $candidate entered unexpected state: ${status:-unknown}" >&2
      if ! delete_replacement "$candidate" "$token"; then
        echo "WARNING: retaining temporary machine image $image for recovery." >&2
        return 1
      fi
      delete_machine_image "$image" || true
      return 1
    fi

    echo "==> Replacement is RUNNING; verifying its authenticated model endpoint"
    if ! verify_remote_candidate "$candidate"; then
      echo "ERROR: replacement verification failed; rolling it back." >&2
      if ! delete_replacement "$candidate" "$token"; then
        echo "WARNING: retaining temporary machine image $image for recovery." >&2
        return 1
      fi
      delete_machine_image "$image" || true
      return 1
    fi

    echo "==> Replacement verified; deleting stopped source VM in $source_zone"
    if ! output="$(gcloud compute instances delete "$INSTANCE" \
        --zone="$source_zone" --quiet 2>&1)"; then
      echo "$output" >&2
      echo "ERROR: source deletion failed; rolling back the replacement in $candidate." >&2
      if ! delete_replacement "$candidate" "$token"; then
        echo "WARNING: retaining temporary machine image $image for recovery." >&2
        return 1
      fi
      delete_machine_image "$image" || true
      return 1
    fi

    ZONE="$candidate"
    delete_machine_image "$image" || true
    echo "==> VM relocated to $ZONE and started."
    return 0
  done

  delete_machine_image "$image" || true
  echo "ERROR: no zone in [$ZONES] had capacity. The source VM remains stopped in $source_zone; GPU billing is not active." >&2
  return 1
}

tunnel_pids() {
  lsof -tiTCP:"$LOCAL_PORT" -sTCP:LISTEN 2>/dev/null | sort -u || true
}

stop_tunnel() {
  local pid command
  while read -r pid; do
    [[ -n "$pid" ]] || continue
    command="$(basename "$(ps -p "$pid" -o comm= | xargs)")"
    if [[ "$command" == "ssh" ]]; then
      kill "$pid" 2>/dev/null || true
    else
      echo "Refusing to stop non-SSH process $pid using local port $LOCAL_PORT ($command)." >&2
      return 1
    fi
  done < <(tunnel_pids)
}

vm_status() {
  resolve_zone || return 1
  [[ -n "$ZONE" ]] || return
  gcloud compute instances describe "$INSTANCE" \
    --zone="$ZONE" --format='value(status)' 2>/dev/null || true
}

start_vm() {
  local status output source_zone
  resolve_zone || return 1
  if [[ -z "$ZONE" ]]; then
    echo "VM $INSTANCE was not found. Run provision_gcp.sh first." >&2
    return 1
  fi
  status="$(vm_status)"
  case "$status" in
    RUNNING) return ;;
    TERMINATED|STOPPED) ;;
    *)
      echo "VM $INSTANCE is in unexpected state: $status" >&2
      return 1
      ;;
  esac

  if output="$(gcloud compute instances start "$INSTANCE" --zone="$ZONE" 2>&1)"; then
    return
  fi
  echo "$output" >&2
  if ! is_capacity_error "$output"; then
    return 1
  fi

  source_zone="$ZONE"
  echo "==> $source_zone has no $INSTANCE GPU capacity; trying safe cross-zone relocation"
  relocate_vm "$source_zone"
}

start_tunnel_for_zone() {
  local zone="$1"
  local existing pid command
  existing="$(tunnel_pids)"
  if [[ -n "$existing" ]]; then
    while read -r pid; do
      command="$(basename "$(ps -p "$pid" -o comm= | xargs)")"
      if [[ "$command" != "ssh" ]]; then
        echo "Local port $LOCAL_PORT is already used by $command (PID $pid)." >&2
        exit 1
      fi
    done <<< "$existing"
    echo "SSH tunnel is already listening on localhost:$LOCAL_PORT."
    return
  fi

  for _ in $(seq 1 30); do
    if gcloud compute ssh "$INSTANCE" --zone="$zone" --command=true >/dev/null 2>&1; then
      break
    fi
    sleep 5
  done

  gcloud compute ssh "$INSTANCE" --zone="$zone" -- \
    -N -f \
    -L "$LOCAL_PORT:localhost:$REMOTE_PORT" \
    -o ExitOnForwardFailure=yes \
    -o ServerAliveInterval=30 \
    -o ServerAliveCountMax=3
}

start_tunnel() {
  start_tunnel_for_zone "$ZONE"
}

verify_tunnel() {
  local token
  token="$(sed -n 's/^AI_API_KEY=//p' "$ENV_FILE" 2>/dev/null || true)"
  if [[ -z "$token" ]]; then
    echo "Tunnel started, but AI_API_KEY is missing from $ENV_FILE; skipping API verification."
    return
  fi
  curl -fsS --max-time 15 "http://127.0.0.1:$LOCAL_PORT/v1/models" \
    -H "Authorization: Bearer $token" >/dev/null
  echo "AI VM is ready at http://localhost:$LOCAL_PORT/v1."
}

case "${1:-status}" in
  start)
    start_vm
    start_tunnel
    verify_tunnel
    ;;
  stop)
    stop_tunnel
    if [[ "$(vm_status)" == "RUNNING" ]]; then
      gcloud compute instances stop "$INSTANCE" --zone="$ZONE"
    else
      echo "VM is already stopped."
    fi
    ;;
  status)
    vm_state="$(vm_status)"
    echo "VM: ${INSTANCE} (${ZONE:-unknown zone}) — ${vm_state:-not found}"
    if [[ -n "$(tunnel_pids)" ]]; then
      echo "Tunnel: listening on localhost:$LOCAL_PORT"
    else
      echo "Tunnel: stopped"
    fi
    ;;
  *)
    echo "Usage: $0 {start|stop|status}" >&2
    exit 2
    ;;
esac
