#!/bin/bash
# Network tuning for 100Mbps Ethernet on ARM

echo "Tuning network for ARM..."

# Optimize for 100Mbps Ethernet
if command -v ethtool &> /dev/null; then
    sudo ethtool -C eth0 rx-usecs 30 tx-usecs 30 2>/dev/null || true
    sudo ethtool -K eth0 gro on gso on tso on 2>/dev/null || true
    echo "Network tuning applied"
else
    echo "ethtool not found, skipping network tuning"
fi
