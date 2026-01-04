#!/bin/bash
# Test memory pressure

echo "Testing memory limits..."

# Check available memory
free -h

# Check tmpfs usage
if mountpoint -q /dev/shm/videos; then
    echo ""
    echo "tmpfs usage:"
    df -h /dev/shm/videos
else
    echo "tmpfs not mounted"
fi

# Simulate memory pressure (optional)
# Uncomment to test OOM handling
# stress-ng --vm 2 --vm-bytes 3G --timeout 30s
