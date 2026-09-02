#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANAGE_SCRIPT="$(cd "$SCRIPT_DIR/.." && pwd)/manage.sh"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/manage-failover-test.XXXXXX")"
trap 'rm -rf "$TEST_ROOT"' EXIT

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

zone_arg() {
  local previous="" argument
  for argument in "$@"; do
    if [[ "$previous" == "--zone" ]]; then
      printf '%s\n' "$argument"
      return
    fi
    case "$argument" in
      --zone=*)
        printf '%s\n' "${argument#--zone=}"
        return
        ;;
    esac
    previous="$argument"
  done
}

gcloud() {
  local zone argument label=""
  printf '%s\n' "$*" >> "$MOCK_STATE/commands.log"

  case "$1 $2 $3" in
    "compute instances list")
      for argument in "$@"; do
        case "$argument" in
          --zones=*)
            zone="${argument#--zones=}"
            ;;
        esac
      done
      if [[ -n "${zone:-}" ]]; then
        if [[ "$zone" == "us-central1-a" &&
            -f "$MOCK_STATE/target-created" && ! -f "$MOCK_STATE/target-deleted" ]]; then
          if [[ "$*" == *"csv[no-heading]"* ]]; then
            [[ -f "$MOCK_STATE/target-label" ]] && label="$(cat "$MOCK_STATE/target-label")"
            printf 'RUNNING,%s\n' "$label"
          else
            printf '%s\n' "ai-model-server"
          fi
        else
          printf '%s\n' "WARNING: filter keys were not present in any resource" >&2
        fi
        return
      fi
      if [[ -f "$MOCK_STATE/source-deleted" ]]; then
        [[ -f "$MOCK_STATE/target-created" && ! -f "$MOCK_STATE/target-deleted" ]] && \
          printf '%s\n' "us-central1-a"
      else
        printf '%s\n' "us-central1-b"
      fi
      ;;
    "compute instances describe")
      zone="$(zone_arg "$@")"
      if [[ "$zone" == "us-central1-b" && ! -f "$MOCK_STATE/source-deleted" ]]; then
        printf '%s\n' "TERMINATED"
      elif [[ "$zone" == "us-central1-a" &&
          -f "$MOCK_STATE/target-created" && ! -f "$MOCK_STATE/target-deleted" ]]; then
        printf '%s\n' "RUNNING"
      else
        return 1
      fi
      ;;
    "compute instances start")
      printf '%s\n' "ZONE_RESOURCE_POOL_EXHAUSTED: stockout" >&2
      return 1
      ;;
    "compute instances create")
      for argument in "$@"; do
        case "$argument" in
          --labels=medical_transcription_relocation=*)
            label="${argument#--labels=medical_transcription_relocation=}"
            ;;
        esac
      done
      if [[ "${MOCK_AMBIGUOUS_CREATE:-0}" == "1" ]]; then
        touch "$MOCK_STATE/target-created"
        printf '%s\n' "another-owner" > "$MOCK_STATE/target-label"
        printf '%s\n' "simulated ambiguous create failure" >&2
        return 1
      fi
      if [[ "${MOCK_TARGET_CAPACITY_FAIL:-0}" == "1" ]]; then
        printf '%s\n' "ZONE_RESOURCE_POOL_EXHAUSTED: stockout" >&2
        return 1
      fi
      touch "$MOCK_STATE/target-created"
      printf '%s\n' "$label" > "$MOCK_STATE/target-label"
      ;;
    "compute instances stop")
      zone="$(zone_arg "$@")"
      [[ "$zone" == "us-central1-a" ]] && touch "$MOCK_STATE/target-stopped"
      ;;
    "compute instances delete")
      zone="$(zone_arg "$@")"
      if [[ "$zone" == "us-central1-b" ]]; then
        if [[ "${MOCK_SOURCE_DELETE_FAIL:-0}" == "1" ]]; then
          printf '%s\n' "simulated source deletion failure" >&2
          return 1
        fi
        touch "$MOCK_STATE/source-deleted"
      else
        if [[ "${MOCK_TARGET_DELETE_FAIL:-0}" == "1" ]]; then
          printf '%s\n' "simulated replacement deletion failure" >&2
          return 1
        fi
        touch "$MOCK_STATE/target-deleted"
      fi
      ;;
    "compute machine-images create")
      if [[ "${MOCK_IMAGE_CREATE_FAIL:-0}" == "1" ]]; then
        printf '%s\n' "simulated image creation failure" >&2
        return 1
      fi
      touch "$MOCK_STATE/image-created"
      ;;
    "compute machine-images delete")
      touch "$MOCK_STATE/image-deleted"
      ;;
    "compute ssh "*)
      if [[ "$*" == *"/v1/models"* ]]; then
        if [[ "${MOCK_VERIFICATION_FAIL:-0}" == "1" ]]; then
          return 1
        fi
        touch "$MOCK_STATE/replacement-verified"
      fi
      ;;
    *)
      fail "unexpected gcloud command: $*"
      ;;
  esac
}

lsof() {
  return 1
}

curl() {
  return 0
}

export -f fail zone_arg gcloud lsof curl

new_case() {
  MOCK_STATE="$TEST_ROOT/$1"
  mkdir -p "$MOCK_STATE"
  : > "$MOCK_STATE/commands.log"
  export MOCK_STATE
  unset MOCK_AMBIGUOUS_CREATE MOCK_IMAGE_CREATE_FAIL MOCK_SOURCE_DELETE_FAIL
  unset MOCK_TARGET_CAPACITY_FAIL MOCK_TARGET_DELETE_FAIL MOCK_VERIFICATION_FAIL
}

assert_file() {
  [[ -f "$1" ]] || fail "expected file: $1"
}

assert_no_file() {
  [[ ! -e "$1" ]] || fail "unexpected file: $1"
}

new_case success
ZONE=us-central1-b ZONES="us-central1-b us-central1-a" \
  bash "$MANAGE_SCRIPT" start > "$MOCK_STATE/output.log" 2>&1
assert_file "$MOCK_STATE/source-deleted"
assert_file "$MOCK_STATE/image-deleted"
assert_file "$MOCK_STATE/target-created"
assert_file "$MOCK_STATE/replacement-verified"
assert_no_file "$MOCK_STATE/target-deleted"
grep -q 'source-machine-image=' "$MOCK_STATE/commands.log" || \
  fail "replacement was not created from a machine image"
grep -Fq 'Authorization: Bearer $token' "$MOCK_STATE/commands.log" || \
  fail "remote verification did not keep the API key out of command arguments"
if grep -q 'instances move' "$MOCK_STATE/commands.log"; then
  fail "retired instances move command was used"
fi

new_case source_delete_rollback
export MOCK_SOURCE_DELETE_FAIL=1
if ZONE=us-central1-b ZONES="us-central1-b us-central1-a" \
    bash "$MANAGE_SCRIPT" start > "$MOCK_STATE/output.log" 2>&1; then
  fail "source deletion failure unexpectedly succeeded"
fi
assert_no_file "$MOCK_STATE/source-deleted"
assert_file "$MOCK_STATE/target-stopped"
assert_file "$MOCK_STATE/target-deleted"
assert_file "$MOCK_STATE/image-deleted"

new_case capacity_rollback
export MOCK_TARGET_CAPACITY_FAIL=1
if ZONE=us-central1-b ZONES="us-central1-b us-central1-a" \
    bash "$MANAGE_SCRIPT" start > "$MOCK_STATE/output.log" 2>&1; then
  fail "all-zones capacity failure unexpectedly succeeded"
fi
assert_no_file "$MOCK_STATE/source-deleted"
assert_no_file "$MOCK_STATE/target-created"
assert_file "$MOCK_STATE/image-deleted"

new_case verification_rollback
export MOCK_VERIFICATION_FAIL=1
if ZONE=us-central1-b ZONES="us-central1-b us-central1-a" \
    RELOCATION_VERIFY_ATTEMPTS=1 RELOCATION_VERIFY_DELAY=0 \
    bash "$MANAGE_SCRIPT" start > "$MOCK_STATE/output.log" 2>&1; then
  fail "replacement verification failure unexpectedly succeeded"
fi
assert_no_file "$MOCK_STATE/source-deleted"
assert_file "$MOCK_STATE/target-stopped"
assert_file "$MOCK_STATE/target-deleted"
assert_file "$MOCK_STATE/image-deleted"

new_case ambiguous_create
export MOCK_AMBIGUOUS_CREATE=1
if ZONE=us-central1-b ZONES="us-central1-b us-central1-a" \
    bash "$MANAGE_SCRIPT" start > "$MOCK_STATE/output.log" 2>&1; then
  fail "ambiguous replacement creation unexpectedly succeeded"
fi
assert_no_file "$MOCK_STATE/source-deleted"
assert_no_file "$MOCK_STATE/target-stopped"
assert_no_file "$MOCK_STATE/target-deleted"
assert_no_file "$MOCK_STATE/image-deleted"
grep -q 'retaining temporary machine image' "$MOCK_STATE/output.log" || \
  fail "ambiguous creation did not retain the recovery image"

new_case cleanup_failure
export MOCK_SOURCE_DELETE_FAIL=1
export MOCK_TARGET_DELETE_FAIL=1
if ZONE=us-central1-b ZONES="us-central1-b us-central1-a" \
    bash "$MANAGE_SCRIPT" start > "$MOCK_STATE/output.log" 2>&1; then
  fail "replacement cleanup failure unexpectedly succeeded"
fi
assert_no_file "$MOCK_STATE/source-deleted"
assert_file "$MOCK_STATE/target-stopped"
assert_no_file "$MOCK_STATE/target-deleted"
assert_no_file "$MOCK_STATE/image-deleted"

new_case image_failure
export MOCK_IMAGE_CREATE_FAIL=1
if ZONE=us-central1-b ZONES="us-central1-b us-central1-a" \
    bash "$MANAGE_SCRIPT" start > "$MOCK_STATE/output.log" 2>&1; then
  fail "machine image creation failure unexpectedly succeeded"
fi
assert_no_file "$MOCK_STATE/source-deleted"
assert_no_file "$MOCK_STATE/target-created"

new_case region_rejection
if ZONE=us-central1-b ZONES="us-central1-b us-east1-c" \
    bash "$MANAGE_SCRIPT" start > "$MOCK_STATE/output.log" 2>&1; then
  fail "cross-region relocation unexpectedly succeeded"
fi
assert_no_file "$MOCK_STATE/source-deleted"
assert_no_file "$MOCK_STATE/target-created"
assert_file "$MOCK_STATE/image-deleted"

printf 'PASS: manage.sh cross-zone failover and rollback scenarios\n'
