#!/usr/bin/env bash
# Nightly backup of the SQLite database and uploads. Run from cron as root:
#   15 3 * * * /usr/local/bin/bkkguide-backup.sh
# Copy BACKUP_DIR off the box (rsync, rclone, ...) for a real off-site backup.
set -euo pipefail

STATE=/var/lib/bkkguide
BACKUP_DIR=${BACKUP_DIR:-/var/backups/bkkguide}
KEEP_DAYS=${KEEP_DAYS:-14}
STAMP=$(date +%Y%m%d-%H%M%S)

mkdir -p "$BACKUP_DIR"
sqlite3 "$STATE/bkkguide.db" ".backup '$BACKUP_DIR/bkkguide-$STAMP.db'"
tar -czf "$BACKUP_DIR/uploads-$STAMP.tar.gz" -C "$STATE" uploads 2>/dev/null || true
find "$BACKUP_DIR" -type f -mtime +"$KEEP_DAYS" -delete
