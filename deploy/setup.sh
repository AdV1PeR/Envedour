#!/bin/bash
set -e

echo "Setting up ARM-optimized Envedour Bot..."

# Install ARM-optimized dependencies
sudo apt-get update
sudo apt-get install -y \
    ffmpeg \
    aria2 \
    python3 \
    python3-pip \
    redis-server \
    build-essential \
    curl \
    wget

# Install yt-dlp with curl-cffi support for TikTok impersonation
sudo pip3 install --upgrade "yt-dlp[default,curl-cffi]"

# Enable hardware acceleration (only if running as non-root user)
if [ "$EUID" -ne 0 ] && [ -n "$SUDO_USER" ]; then
    sudo usermod -a -G video "$SUDO_USER"
elif [ "$EUID" -ne 0 ]; then
    sudo usermod -a -G video "$USER"
fi

# Create tmpfs directory
sudo mkdir -p /dev/shm/videos
sudo chmod 1777 /dev/shm/videos

# Setup Redis
sudo systemctl enable redis-server
sudo systemctl start redis-server

# Configure Redis for ARM
# Create directory if it doesn't exist
sudo mkdir -p /etc/redis/redis.conf.d

sudo tee /etc/redis/redis.conf.d/arm-optimized.conf > /dev/null <<EOF
maxmemory 1gb
maxmemory-policy allkeys-lru
save ""
activerehashing yes
EOF

sudo systemctl restart redis-server

echo "Setup complete!"
