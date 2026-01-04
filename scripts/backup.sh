#!/bin/bash
# Backup script for Redis data

BACKUP_DIR="/opt/backups/envedour-bot"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/redis_backup_$DATE.rdb"

mkdir -p "$BACKUP_DIR"

echo "Creating Redis backup..."

# Check if redis-cli is available
if ! command -v redis-cli &> /dev/null; then
    echo "Error: redis-cli not found"
    exit 1
fi

# Trigger Redis save
redis-cli BGSAVE > /dev/null 2>&1

# Wait for save to complete (check if LASTSAVE changed)
LAST_SAVE=$(redis-cli LASTSAVE)
sleep 2
while [ "$LAST_SAVE" = "$(redis-cli LASTSAVE)" ]; do
    sleep 1
done

# Copy RDB file
if [ -f /var/lib/redis/dump.rdb ]; then
    cp /var/lib/redis/dump.rdb "$BACKUP_FILE"
    echo "Backup created: $BACKUP_FILE"
    
    # Keep only last 7 days of backups
    find "$BACKUP_DIR" -name "redis_backup_*.rdb" -mtime +7 -delete
else
    echo "Error: Redis dump file not found"
    exit 1
fi
