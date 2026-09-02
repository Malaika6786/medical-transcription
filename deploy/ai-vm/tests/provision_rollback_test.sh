#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROVISION_SCRIPT="$(cd "$SCRIPT_DIR/.." && pwd)/provision_gcp.sh"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/provision-rollback-test.XXXXXX")"
trap 'rm -rf "$TEST_ROOT"' EXIT

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

argument_value() {
  local key="$1" argument
  shift
  for argument in "$@"; do
    case "$argument" in
      "$key"=*)
        printf '%s\n' "${argument#"$key"=}"
        return
        ;;
    esac
  done
}

gcloud() {
  local zone label command_text="$*"
  printf '%s\n' "$command_text" >> "$MOCK_STATE/commands.log"

  case "$1 $2 $3" in
    "compute instances list")
      zone="$(argument_value --zones "$@")"
      if [[ -f "$MOCK_STATE/created-$zone" && ! -f "$MOCK_STATE/deleted-$zone" ]]; then
        if [[ "$command_text" == *"csv[no-heading]"* ]]; then
          label="$(cat "$MOCK_STATE/label-$zone")"
          printf 'RUNNING,%s\n' "$label"
        else
          printf '%s\n' "ai-model-server"
        fi
      else
        printf '%s\n' "WARNING: filter keys were not present in any resource" >&2
      fi
      ;;
    "compute instances create")
      zone="$(argument_value --zone "$@")"
      label="$(argument_value --labels "$@")"
      label="${label#medical_transcription_provision=}"
      touch "$MOCK_STATE/created-$zone"
      printf '%s\n' "$label" > "$MOCK_STATE/label-$zone"
      ;;
    "compute instances stop")
      zone="$(argument_value --zone "$@")"
      touch "$MOCK_STATE/stopped-$zone"
      ;;
    "compute instances delete")
      zone="$(argument_value --zone "$@")"
      touch "$MOCK_STATE/deleted-$zone"
      ;;
    "compute ssh "*)
      if [[ "$command_text" == *"--command=true"* ]]; then
        [[ "${MOCK_SSH_FAIL:-0}" == "0" ]]
      elif [[ "$command_text" == *"setup.sh"* ]]; then
        [[ "${MOCK_SETUP_FAIL:-0}" == "0" ]]
      fi
      ;;
    "compute scp "*)
      [[ "${MOCK_SCP_FAIL:-0}" == "0" ]]
      ;;
    *)
      fail "unexpected gcloud command: $command_text"
      ;;
  esac
}

sleep() {
  :
}

export -f fail argument_value gcloud sleep

new_case() {
  MOCK_STATE="$TEST_ROOT/$1"
  mkdir -p "$MOCK_STATE"
  : > "$MOCK_STATE/commands.log"
  export MOCK_STATE
  unset MOCK_SCP_FAIL MOCK_SETUP_FAIL MOCK_SSH_FAIL
}

assert_file() {
  [[ -f "$1" ]] || fail "expected file: $1"
}

assert_no_file() {
  [[ ! -e "$1" ]] || fail "unexpected file: $1"
}

new_case success
ZONE=us-central1-c bash "$PROVISION_SCRIPT" > "$MOCK_STATE/output.log" 2>&1
assert_file "$MOCK_STATE/created-us-central1-c"
assert_no_file "$MOCK_STATE/stopped-us-central1-c"
assert_no_file "$MOCK_STATE/deleted-us-central1-c"
if grep -q -- '--zone=us-central1-b' "$MOCK_STATE/commands.log"; then
  fail "ZONE override was ignored"
fi

new_case setup_failure
export MOCK_SETUP_FAIL=1
if ZONE=us-central1-a bash "$PROVISION_SCRIPT" > "$MOCK_STATE/output.log" 2>&1; then
  fail "setup failure unexpectedly succeeded"
fi
assert_file "$MOCK_STATE/created-us-central1-a"
assert_file "$MOCK_STATE/stopped-us-central1-a"
assert_file "$MOCK_STATE/deleted-us-central1-a"

new_case ssh_failure
export MOCK_SSH_FAIL=1
if ZONE=us-central1-f bash "$PROVISION_SCRIPT" > "$MOCK_STATE/output.log" 2>&1; then
  fail "SSH readiness failure unexpectedly succeeded"
fi
assert_file "$MOCK_STATE/created-us-central1-f"
assert_file "$MOCK_STATE/stopped-us-central1-f"
assert_file "$MOCK_STATE/deleted-us-central1-f"

printf 'PASS: provision_gcp.sh success and rollback scenarios\n'
