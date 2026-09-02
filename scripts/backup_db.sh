#!/usr/bin/env bash
# Back up the medical-transcription PostgreSQL database (meditrans).
#
#   bash scripts/backup_db.sh                 # custom-format dump -> backups/
#   bash scripts/backup_db.sh --plain         # additionally write a readable .sql
#   bash scripts/backup_db.sh --out /some/dir # write somewhere else
#   bash scripts/backup_db.sh --verify        # restore into a scratch DB and compare row counts
#
# Connection is taken from DATABASE_URL, then backend/.env, then the local
# default (postgres://localhost:5432/meditrans?sslmode=disable).
#
# The dump contains patient transcripts and bcrypt password hashes. Keep it out
# of git (backups/ is gitignored) and off shared drives.
#
# Restore: see scripts/restore_db.sh or docs/DATABASE.md.
set -Eeuo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="$REPO_ROOT/backups"
PLAIN=0
VERIFY=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --plain)   PLAIN=1; shift ;;
    --verify)  VERIFY=1; shift ;;
    --out)     OUT_DIR="${2:?--out needs a directory}"; shift 2 ;;
    -h|--help) sed -n '2,17p' "${BASH_SOURCE[0]}"; exit 0 ;;
    *)         echo "Unknown argument: $1" >&2; exit 2 ;;
  esac
done

# --- connection -------------------------------------------------------------

env_value() {
  sed -n "s/^${1}=//p" "$REPO_ROOT/backend/.env" 2>/dev/null | tail -n 1
}

DB_URL="${DATABASE_URL:-$(env_value DATABASE_URL)}"
DB_URL="${DB_URL:-postgres://localhost:5432/meditrans?sslmode=disable}"

# Split "postgres://host:port/dbname?opts" into its parts with parameter
# expansion — BSD sed has no \| alternation, and getting this wrong once meant
# restoring a dump over the live database.
url_query() { case "$1" in *\?*) printf '?%s' "${1#*\?}" ;; esac; }
url_dbname() { local u="${1%%\?*}"; printf '%s' "${u##*/}"; }
url_with_db() {
  local u="${1%%\?*}"
  printf '%s/%s%s' "${u%/*}" "$2" "$(url_query "$1")"
}

DB_NAME="$(url_dbname "$DB_URL")"
if [[ -z "$DB_NAME" || "$(url_with_db "$DB_URL" "$DB_NAME")" != "$DB_URL" ]]; then
  echo "ERROR: DATABASE_URL must be a URI ending in a database name," >&2
  echo "       e.g. postgres://localhost:5432/meditrans?sslmode=disable" >&2
  echo "       Got: $DB_URL" >&2
  exit 1
fi

for binary in pg_dump psql; do
  command -v "$binary" >/dev/null 2>&1 || {
    echo "ERROR: $binary not found on PATH." >&2
    echo "  macOS: brew install postgresql@17 && brew link --force postgresql@17" >&2
    exit 1
  }
done

if ! psql "$DB_URL" -Atc 'SELECT 1' >/dev/null 2>&1; then
  echo "ERROR: cannot connect to $DB_URL" >&2
  echo "  Start PostgreSQL first: brew services start postgresql@17" >&2
  exit 1
fi

# pg_dump refuses to run against a newer server than itself.
SERVER_VERSION="$(psql "$DB_URL" -Atc 'SHOW server_version' | awk '{print $1}')"
DUMP_VERSION="$(pg_dump --version | awk '{print $3}')"

mkdir -p "$OUT_DIR"
chmod 700 "$OUT_DIR"

STAMP="$(date +%Y%m%d-%H%M%S)"
BASE="$OUT_DIR/${DB_NAME}-${STAMP}"

# --- dump -------------------------------------------------------------------

echo "Database : $DB_NAME (server $SERVER_VERSION, pg_dump $DUMP_VERSION)"
echo "Output   : $BASE.dump"

# -Fc  custom format: compressed, restorable selectively with pg_restore
# --no-owner / --no-privileges: restorable under any local role, since the dev
#   machine owns the objects as the OS user and Docker owns them as "meditrans"
pg_dump "$DB_URL" \
  --format=custom \
  --no-owner \
  --no-privileges \
  --file="$BASE.dump"
chmod 600 "$BASE.dump"

if [[ "$PLAIN" == '1' ]]; then
  pg_dump "$DB_URL" --format=plain --no-owner --no-privileges --file="$BASE.sql"
  chmod 600 "$BASE.sql"
  echo "Plain SQL: $BASE.sql"
fi

# Row counts of the tables that matter, recorded next to the dump so a restore
# can be checked without guessing what "complete" looked like.
psql "$DB_URL" -Atc "
  SELECT 'users',             count(*) FROM users
  UNION ALL SELECT 'roles',             count(*) FROM roles
  UNION ALL SELECT 'sessions',          count(*) FROM sessions
  UNION ALL SELECT 'transcript_chunks', count(*) FROM transcript_chunks
  UNION ALL SELECT 'corti_templates',   count(*) FROM corti_templates
  ORDER BY 1" > "$BASE.manifest.txt"
{
  echo "database=$DB_NAME"
  echo "server_version=$SERVER_VERSION"
  echo "pg_dump_version=$DUMP_VERSION"
  echo "pgvector=$(psql "$DB_URL" -Atc "SELECT extversion FROM pg_extension WHERE extname='vector'")"
  echo "taken_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
} >> "$BASE.manifest.txt"
chmod 600 "$BASE.manifest.txt"

echo "Manifest : $BASE.manifest.txt"
echo
cat "$BASE.manifest.txt"
echo

# --- optional restore check -------------------------------------------------

if [[ "$VERIFY" == '1' ]]; then
  SCRATCH="${DB_NAME}_restorecheck_$(date +%s)"
  ADMIN_URL="$(url_with_db "$DB_URL" postgres)"
  RESTORE_URL="$(url_with_db "$DB_URL" "$SCRATCH")"

  # Never let a verification restore touch the database being backed up.
  if [[ "$RESTORE_URL" == "$DB_URL" || "$(url_dbname "$RESTORE_URL")" != "$SCRATCH" ]]; then
    echo "ERROR: could not derive a distinct scratch database URL; skipping --verify." >&2
    exit 1
  fi

  echo "Verifying restore into scratch database $SCRATCH ..."
  cleanup() { psql "$ADMIN_URL" -q -c "DROP DATABASE IF EXISTS \"$SCRATCH\"" >/dev/null 2>&1 || true; }
  trap cleanup EXIT

  psql "$ADMIN_URL" -q -c "CREATE DATABASE \"$SCRATCH\""
  pg_restore --dbname="$RESTORE_URL" --no-owner --no-privileges "$BASE.dump"

  psql "$RESTORE_URL" -Atc "
    SELECT 'users',             count(*) FROM users
    UNION ALL SELECT 'roles',             count(*) FROM roles
    UNION ALL SELECT 'sessions',          count(*) FROM sessions
    UNION ALL SELECT 'transcript_chunks', count(*) FROM transcript_chunks
    UNION ALL SELECT 'corti_templates',   count(*) FROM corti_templates
    ORDER BY 1" > "$OUT_DIR/.verify-$STAMP"

  if diff <(grep -E '^[a-z_]+\|' "$BASE.manifest.txt") "$OUT_DIR/.verify-$STAMP" >/dev/null; then
    echo "Restore verified: row counts match the source database."
  else
    echo "ERROR: restored row counts differ from the source:" >&2
    diff <(grep -E '^[a-z_]+\|' "$BASE.manifest.txt") "$OUT_DIR/.verify-$STAMP" >&2 || true
    rm -f "$OUT_DIR/.verify-$STAMP"
    exit 1
  fi
  rm -f "$OUT_DIR/.verify-$STAMP"
fi

echo "Done. Restore with: bash scripts/restore_db.sh $BASE.dump"
