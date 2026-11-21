#!/bin/sh
# Dump HMSMaster's PostgreSQL database into ./backups and prune old dumps.
set -eu
cd "$(dirname "$0")/.."
mkdir -p backups
STAMP=$(date +%Y-%m-%d_%H%M%S)
OUT="backups/hmsmaster_${STAMP}.sql.gz"
docker compose exec -T db sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" pg_dump -U hms_user -d hmsmaster' | gzip > "$OUT"
test -s "$OUT"
find backups -name 'hmsmaster_*.sql.gz' -mtime +"${HMS_BACKUP_RETENTION_DAYS:-14}" -delete
echo "BACKUP_OK $OUT $(du -h "$OUT" | cut -f1)"
