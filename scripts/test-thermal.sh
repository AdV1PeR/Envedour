#!/bin/bash
# Test thermal throttling with stress-ng

echo "Testing thermal management..."

# Check current temperature
if [ -f /sys/class/thermal/thermal_zone0/temp ]; then
    temp=$(cat /sys/class/thermal/thermal_zone0/temp)
    temp_c=$((temp / 1000))
    echo "Current temperature: ${temp_c}°C"
else
    echo "Thermal zone not available"
    exit 1
fi

# Check if stress-ng is available
if ! command -v stress-ng &> /dev/null; then
    echo "Error: stress-ng not found. Install it with: sudo apt-get install stress-ng"
    exit 1
fi

echo "Starting stress test (will run for 60 seconds)..."
echo "Monitor temperature with: watch -n 1 'cat /sys/class/thermal/thermal_zone0/temp'"

# Run stress test
stress-ng --cpu 4 --timeout 60s

echo "Stress test complete"
temp=$(cat /sys/class/thermal/thermal_zone0/temp)
temp_c=$((temp / 1000))
echo "Final temperature: ${temp_c}°C"
