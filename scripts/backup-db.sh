#!/usr/bin/env bash
#
# Full database backup for kitamanager.
#
# Reads connection details from environment variables (same as the API server)
# or falls back to .env / .env.example defaults.
#
# Usage:
#   ./scripts/backup-db.sh                        # writes to backups/<timestamp>.sql.gz
#   ./scripts/backup-db.sh /path/to/output.sql.gz # writes to specified path
#
# For docker-compose deployments:
#   DB_HOST=localhost DB_PORT=5432 ./scripts/backup-db.sh

set -euo pipefail

# Load .env if present (does not override already-set vars)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

if [[ -f "$PROJECT_DIR/.env" ]]; then
    set -a
    # shellcheck source=/dev/null
    source "$PROJECT_DIR/.env"
    set +a
fi

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-kitamanager}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_NAME="${DB_NAME:-kitamanager}"
DB_SSLMODE="${DB_SSLMODE:-prefer}"

TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
BACKUP_DIR="$PROJECT_DIR/backups"

if [[ -n "${1:-}" ]]; then
    OUTPUT="$1"
else
    mkdir -p "$BACKUP_DIR"
    OUTPUT="$BACKUP_DIR/${DB_NAME}_${TIMESTAMP}.sql.gz"
fi

echo "Backing up database '$DB_NAME' on ${DB_HOST}:${DB_PORT} ..."

export PGPASSWORD="$DB_PASSWORD"

pg_dump \
    --host="$DB_HOST" \
    --port="$DB_PORT" \
    --username="$DB_USER" \
    --dbname="$DB_NAME" \
    --format=plain \
    --no-owner \
    --no-privileges \
    | gzip > "$OUTPUT"

unset PGPASSWORD

SIZE="$(du -h "$OUTPUT" | cut -f1)"
echo "Backup complete: $OUTPUT ($SIZE)"
