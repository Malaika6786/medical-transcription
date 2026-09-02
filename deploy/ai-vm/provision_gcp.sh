#!/usr/bin/env bash
# Provision a GCP GPU VM for the AI Assistance model server, then run setup.sh
# on it. Run from your laptop (gcloud must be authenticated).
#
#   bash provision_gcp.sh
#
# COST WARNING: g2-standard-4 (1x NVIDIA L4 24GB) is ~\$0.70-0.85/hr on-demand
# (~\$500+/month if left running). Stop it when idle:
#   gcloud compute instances stop ai-model-server --zone=us-central1-b
#
# GPU quota: new projects often have 0 GPU quota. If creation fails with a
# quota error, request "GPUs (all regions)" >= 1 at
# https://console.cloud.google.com/iam-admin/quotas
set -euo pipefail

INSTANCE="${INSTANCE:-ai-model-server}"
MACHINE="${MACHINE:-g2-standard-4}"   # includes 1x L4 24GB
DISK_GB="${DISK_GB:-80}"
REQUESTED_ZONE="${ZONE:-}"
# Zones tried in order; if one is out of L4 capacity the partial instance is
# deleted (rollback) and the next zone is attempted.
ZONES="${ZONES:-${REQUESTED_ZONE:-us-central1-b us-central1-a us-central1-c us-central1-f}}"
PROVISION_LABEL_KEY="medical_transcription_provision"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROVISION_TOKEN="p$(date +%Y%m%d%H%M%S)-$$-$RANDOM"
CREATED_ZONE=""
PROVISION_COMMITTED=0

is_capacity_error() {
  grep -qiE 'ZONE_RESOURCE_POOL_EXHAUSTED|does not have enough resources available|stockout|out of capacity' <<<"$1"
}

provision_probe() {
  local zone="$1"
  local output count actual

  if ! output="$(gcloud compute instances list \
      --zones="$zone" \
      --filter="name=('$INSTANCE')" \
      --format="csv[no-heading](status,labels.$PROVISION_LABEL_KEY)" 2>/dev/null)"; then
    return 2
  fi
  count="$(awk 'NF { count++ } END { print count + 0 }' <<<"$output")"
  [[ "$count" -eq 0 ]] && return 1
  [[ "$count" -eq 1 ]] || return 2
  actual="${output#*,}"
  [[ "$actual" == "$PROVISION_TOKEN" ]] || return 2
  return 0
}

delete_owned_instance() {
  local zone="$1"
  local probe_result output

  if provision_probe "$zone"; then
    :
  else
    probe_result=$?
    [[ "$probe_result" -eq 1 ]] && return 0
    echo "ERROR: refusing to delete $INSTANCE in $zone because ownership could not be proven." >&2
    return 1
  fi

  gcloud compute instances stop "$INSTANCE" --zone="$zone" --quiet >/dev/null 2>&1 || true
  if ! output="$(gcloud compute instances delete "$INSTANCE" \
      --zone="$zone" --delete-disks=all --quiet 2>&1)"; then
    echo "$output" >&2
    echo "ERROR: $INSTANCE in $zone was stopped but could not be deleted; clean it up manually." >&2
    return 1
  fi
}

rollback_on_exit() {
  local exit_code=$?
  trap - EXIT

  if [[ "$exit_code" -ne 0 && "$PROVISION_COMMITTED" == "0" && -n "$CREATED_ZONE" ]]; then
    echo "==> Provisioning failed; rolling back owned VM in $CREATED_ZONE" >&2
    if ! delete_owned_instance "$CREATED_ZONE"; then
      echo "WARNING: automatic rollback was incomplete; check $INSTANCE in $CREATED_ZONE." >&2
      echo "WARNING: the VM may still be RUNNING and incurring GPU billing." >&2
    fi
  fi
  exit "$exit_code"
}

trap rollback_on_exit EXIT

ZONE=""
for candidate in $ZONES; do
  if ! existing="$(gcloud compute instances list \
      --zones="$candidate" \
      --filter="name=('$INSTANCE')" \
      --format='value(name)' 2>/dev/null)"; then
    echo "ERROR: could not determine whether $candidate is safe for provisioning." >&2
    exit 1
  fi
  if [[ -n "$existing" ]]; then
    echo "==> $INSTANCE already exists in $candidate. Use manage.sh start, or delete it first." >&2
    exit 1
  fi

  echo "==> Creating $INSTANCE ($MACHINE, zone $candidate)"
  if output="$(gcloud compute instances create "$INSTANCE" \
    --zone="$candidate" \
    --machine-type="$MACHINE" \
    --image-family=ubuntu-accelerator-2204-amd64-with-nvidia-580 \
    --image-project=ubuntu-os-accelerator-images \
    --boot-disk-size="${DISK_GB}GB" \
    --boot-disk-type=pd-balanced \
    --maintenance-policy=TERMINATE \
    --labels="$PROVISION_LABEL_KEY=$PROVISION_TOKEN" 2>&1)"; then
    ZONE="$candidate"
    CREATED_ZONE="$candidate"
    break
  fi
  echo "$output" >&2
  if provision_probe "$candidate"; then
    CREATED_ZONE="$candidate"
    if ! delete_owned_instance "$candidate"; then
      exit 1
    fi
    CREATED_ZONE=""
  else
    probe_result=$?
    if [[ "$probe_result" -eq 2 ]]; then
      echo "ERROR: instance creation was ambiguous; refusing unproven cleanup in $candidate." >&2
      exit 1
    fi
  fi
  if ! is_capacity_error "$output"; then
    echo "==> Creation failed for a non-capacity reason; not trying other zones." >&2
    exit 1
  fi
  echo "==> No $MACHINE capacity in $candidate — rolling back and trying next zone"
done

if [[ -z "$ZONE" ]]; then
  echo "ERROR: no zone in [$ZONES] had capacity for $MACHINE." >&2
  echo "Try again later, or extend the list, e.g. ZONES=\"us-east1-c us-east4-a\" bash $0" >&2
  exit 1
fi

echo "==> Waiting for SSH to come up"
ssh_ready=0
for i in $(seq 1 30); do
  if gcloud compute ssh "$INSTANCE" --zone="$ZONE" --command='true' 2>/dev/null; then
    ssh_ready=1
    break
  fi
  sleep 10
done
if [[ "$ssh_ready" == "0" ]]; then
  echo "ERROR: SSH did not become ready for $INSTANCE in $ZONE." >&2
  exit 1
fi

echo "==> Copying deploy files and running setup.sh (driver install on first boot can add a few minutes)"
gcloud compute scp --zone="$ZONE" --recurse "$SCRIPT_DIR" "$INSTANCE":~/ai-vm
gcloud compute ssh "$INSTANCE" --zone="$ZONE" --command='sudo bash ~/ai-vm/setup.sh'
PROVISION_COMMITTED=1

cat <<EOF

==> Done. Note the AI_BASE_URL / AI_API_KEY printed above.

The VM's port 8443 is NOT opened to the internet by default (good).
Reach it from the backend via one of:
  - SSH tunnel:  gcloud compute ssh $INSTANCE --zone=$ZONE -- -N -L 8443:localhost:8443
                 (then AI_BASE_URL=http://localhost:8443/v1)
  - Same-VPC private IP (deploy the backend to GCP)
  - Tailscale/WireGuard on the VM

Manage cost:
  stop:   gcloud compute instances stop  $INSTANCE --zone=$ZONE
  start:  gcloud compute instances start $INSTANCE --zone=$ZONE
  delete: gcloud compute instances delete $INSTANCE --zone=$ZONE
EOF
