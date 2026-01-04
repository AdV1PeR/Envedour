#!/bin/bash
# Health check script for monitoring

EXIT_CODE=0

# Check if bot service is running
if ! systemctl is-active --quiet envedour-down-bot; then
    echo "ERROR: Bot service (envedour-down-bot) is not running"
    EXIT_CODE=1
fi

# Check if telegram-bot-api service is running
if ! systemctl is-active --quiet telegram-bot-api; then
    echo "ERROR: Telegram Bot API service is not running"
    EXIT_CODE=1
fi

# Check if Redis is running
if ! systemctl is-active --quiet redis-server; then
    echo "ERROR: Redis is not running"
    EXIT_CODE=1
fi

# Check temperature
if [ -f /sys/class/thermal/thermal_zone0/temp ]; then
    temp=$(cat /sys/class/thermal/thermal_zone0/temp)
    temp_c=$((temp / 1000))
    if [ $temp_c -gt 85 ]; then
        echo "WARNING: High temperature: ${temp_c}°C"
        EXIT_CODE=1
    fi
fi

# Check memory
free_mem=$(free -m | awk 'NR==2{printf "%.0f", $4}')
if [ $free_mem -lt 512 ]; then
    echo "WARNING: Low memory: ${free_mem}MB free"
    EXIT_CODE=1
fi

# Check tmpfs
if ! mountpoint -q /dev/shm/videos; then
    echo "ERROR: tmpfs not mounted"
    EXIT_CODE=1
fi

exit $EXIT_CODE
