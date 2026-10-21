#!/bin/sh
# gitf backup script
# Creates a compressed archive of the data directory.
# Usage: ./scripts/backup.sh [data_dir]

DATA_DIR="${1:-./data}"
BACKUP_FILE="gitf-backup-$(date +%Y%m%d-%H%M%S).tar.gz"

echo "Backing up $DATA_DIR to $BACKUP_FILE..."
tar -czf "$BACKUP_FILE" "$DATA_DIR" && echo "Done: $BACKUP_FILE"
