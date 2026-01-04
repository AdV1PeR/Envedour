#!/bin/bash
# Monitoring script for ARM performance

echo "=== Telegram Bot ARM Performance Monitor ==="
echo ""

echo "Temperature:"
if [ -f /sys/class/thermal/thermal_zone0/temp ]; then
    temp=$(cat /sys/class/thermal/thermal_zone0/temp)
    temp_c=$((temp / 1000))
    echo "  CPU: ${temp_c}°C"
else
    echo "  Thermal zone not available"
fi

echo ""
echo "CPU Frequency:"
for cpu in /sys/devices/system/cpu/cpu*/cpufreq/scaling_cur_freq; do
    if [ -f "$cpu" ]; then
        freq=$(cat "$cpu")
        freq_mhz=$((freq / 1000))
        echo "  $(basename $(dirname $(dirname $cpu))): ${freq_mhz}MHz"
    fi
done

echo ""
echo "Memory:"
free -h

echo ""
echo "Disk (tmpfs):"
df -h /dev/shm/videos 2>/dev/null || echo "  tmpfs not mounted"

echo ""
echo "Redis Status:"
if systemctl is-active --quiet redis-server; then
    echo "  Status: Running"
    redis-cli INFO memory | grep used_memory_human || echo "  Could not get memory info"
else
    echo "  Status: Not running"
fi

echo ""
echo "Bot Services Status:"
if systemctl is-active --quiet envedour-down-bot; then
    echo "  envedour-down-bot: Running"
    systemctl status envedour-down-bot --no-pager -l | head -3
else
    echo "  envedour-down-bot: Not running"
fi

if systemctl is-active --quiet telegram-bot-api; then
    echo "  telegram-bot-api: Running"
    systemctl status telegram-bot-api --no-pager -l | head -3
else
    echo "  telegram-bot-api: Not running"
fi

echo ""
echo "Network:"
if command -v ethtool &> /dev/null; then
    ethtool -S eth0 2>/dev/null | grep -E "(rx_bytes|tx_bytes)" | head -2 || echo "  Network stats not available"
fi
