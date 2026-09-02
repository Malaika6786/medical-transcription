#!/usr/bin/env bash
# Restore a medical-transcription database dump produced by scripts/backup_db.sh.
#
#   bash scripts/restore_db.sh backups/meditrans-20260812-214500.dump
#   bash scripts/restore_db.sh --into meditrans_copy backups/....dump
#
# Default target is the database named in DATABASE_URL (backend/.env, else the
# local default). Restoring into the live database DROPS AND RECREATES it, so
# the script asks for confirmation unless --force is given.
#
# The dump is schema+data. The server re-applies internal/pgstore/schema.sql at
# every startup, so a restored database is immediately usable; pgvector must be
# installed on the target server (CREATE EXTENSION vector is part of the dump).
set -Eeuo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET_DB=""
FORCE=0
DUMP=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --into)    TARGET_DB="${2:?--into needs a database name}"; shift 2 ;;
    --force)   FORCE=1; shift ;;
    -h|--help) sed -n '2,14p' "${BASH_SOURCE[0]}"; exit 0 ;;
    *)         DUMP="$1"; shift ;;
  esac
done

[[ -n "$DUMP" ]] || { echo "Usage: bash scripts/restore_db.sh [--into DB] [--force] <dump-file>" >&2; exit 2; }
[[ -f "$DUMP" ]] || { echo "ERROR: dump file not found: $DUMP" >&2; exit 1; }

env_value() {
  sed -n "s/^${1}=//p" "$REPO_ROOT/backend/.env" 2>/dev/null | tail -n 1
}

DB_URL="${DATABASE_URL:-$(env_value DATABASE_URL)}"
DB_URL="${DB_URL:-postgres://localhost:5432/meditrans?sslmode=disable}"

# Parameter expansion, not sed: BSD sed has no \| alternation, and a URL
# rewrite that silently no-ops here would drop the wrong database.
url_query() { case "$1" in *\?*) printf '?%s' "${1#*\?}" ;; esac; }
url_dbname() { local u="${1%%\?*}"; printf '%s' "${u##*/}"; }
url_with_db() {
  local u="${1%%\?*}"
  printf '%s/%s%s' "${u%/*}" "$2" "$(url_query "$1")"
}

SOURCE_DB="$(url_dbname "$DB_URL")"
if [[ -z "$SOURCE_DB" || "$(url_with_db "$DB_URL" "$SOURCE_DB")" != "$DB_URL" ]]; then
  echo "ERROR: DATABASE_URL must be a URI ending in a database name," >&2
  echo "       e.g. postgres://localhost:5432/meditrans?sslmode=disable" >&2
  echo "       Got: $DB_URL" >&2
  exit 1
fi
TARGET_DB="${TARGET_DB:-$SOURCE_DB}"

for binary in pg_restore psql; do
  command -v "$binary" >/dev/null 2>&1 || { echo "ERROR: $binary not found on PATH." >&2; exit 1; }
done

ADMIN_URL="$(url_with_db "$DB_URL" postgres)"
TARGET_URL="$(url_with_db "$DB_URL" "$TARGET_DB")"
[[ "$(url_dbname "$TARGET_URL")" == "$TARGET_DB" ]] || {
  echo "ERROR: could not build a connection URL for database \"$TARGET_DB\"." >&2
  exit 1
}

psql "$ADMIN_URL" -Atc 'SELECT 1' >/dev/null 2>&1 || {
  echo "ERROR: cannot reach the PostgreSQL server via $ADMIN_URL" >&2
  exit 1
}

echo "Dump   : $DUMP"
echo "Target : $TARGET_DB (server reached via $ADMIN_URL)"

if [[ "$FORCE" != '1' ]]; then
  read -r -p "This DROPS database \"$TARGET_DB\" and recreates it from the dump. Continue? [y/N] " reply
  [[ "$reply" == 'y' || "$reply" == 'Y' ]] || { echo "Aborted."; exit 1; }
fi

# Stop the backend first, or these connections keep the DROP from succeeding.
psql "$ADMIN_URL" -q -c \
  "SELECT pg_terminate_backend(pid) FROM pg_stat_activity
    WHERE datname = '$TARGET_DB' AND pid <> pg_backend_pid()" >/dev/null

psql "$ADMIN_URL" -q -c "DROP DATABASE IF EXISTS \"$TARGET_DB\""
psql "$ADMIN_URL" -q -c "CREATE DATABASE \"$TARGET_DB\""

pg_restore --dbname="$TARGET_URL" --no-owner --no-privileges "$DUMP"

echo
psql "$TARGET_URL" -Atc "
  SELECT 'users',             count(*) FROM users
  UNION ALL SELECT 'roles',             count(*) FROM roles
  UNION ALL SELECT 'sessions',          count(*) FROM sessions
  UNION ALL SELECT 'transcript_chunks', count(*) FROM transcript_chunks
  UNION ALL SELECT 'corti_templates',   count(*) FROM corti_templates
  ORDER BY 1"

MANIFEST="${DUMP%.dump}.manifest.txt"
if [[ -f "$MANIFEST" ]]; then
  echo
  echo "Expected (from $MANIFEST):"
  grep -E '^[a-z_]+\|' "$MANIFEST"
fi

echo
echo "Restore complete. Start the backend; it re-applies the schema and backfills any missing embeddings."
