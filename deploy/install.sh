#!/bin/bash
set -e

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Installing Telegram Bot services..."
echo "Project directory: $PROJECT_DIR"

# Create user if doesn't exist
if ! id "botuser" &>/dev/null; then
    sudo useradd -r -s /bin/false botuser
fi

# Create application directories
sudo mkdir -p /opt/envedour-bot
sudo mkdir -p /opt/telegram-bot-api/data
sudo chown -R botuser:botuser /opt/envedour-bot
sudo chown -R botuser:botuser /opt/telegram-bot-api

# Copy binary
if [ -f "$PROJECT_DIR/envedour-bot-arm64" ]; then
    sudo cp "$PROJECT_DIR/envedour-bot-arm64" /opt/envedour-bot/
    sudo chmod +x /opt/envedour-bot/envedour-bot-arm64
    sudo chown botuser:botuser /opt/envedour-bot/envedour-bot-arm64
    echo "Binary copied to /opt/envedour-bot/envedour-bot-arm64"
else
    echo "Warning: envedour-bot-arm64 not found. Please build it first: make build-arm"
fi

# Copy .env file if exists
if [ -f "$PROJECT_DIR/.env" ]; then
    sudo cp "$PROJECT_DIR/.env" /opt/envedour-bot/.env
    sudo chown botuser:botuser /opt/envedour-bot/.env
    sudo cp "$PROJECT_DIR/.env" /opt/telegram-bot-api/.env
    sudo chown botuser:botuser /opt/telegram-bot-api/.env
    echo ".env file copied to /opt/envedour-bot/.env and /opt/telegram-bot-api/.env"
    
    # Create cookies files if they don't exist and set proper permissions
    # Read paths from .env file
    if [ -f "/opt/envedour-bot/.env" ]; then
        TIKTOK_COOKIES=$(grep "^TIKTOK_COOKIES=" /opt/envedour-bot/.env | cut -d'=' -f2 | tr -d '"' | tr -d "'")
        INSTAGRAM_COOKIES=$(grep "^INSTAGRAM_COOKIES=" /opt/envedour-bot/.env | cut -d'=' -f2 | tr -d '"' | tr -d "'")
        YOUTUBE_COOKIES=$(grep "^YOUTUBE_COOKIES=" /opt/envedour-bot/.env | cut -d'=' -f2 | tr -d '"' | tr -d "'")
        
        for COOKIES_FILE in "$TIKTOK_COOKIES" "$INSTAGRAM_COOKIES" "$YOUTUBE_COOKIES"; do
            if [ -n "$COOKIES_FILE" ]; then
                # Create directory if needed
                COOKIES_DIR=$(dirname "$COOKIES_FILE")
                if [ ! -d "$COOKIES_DIR" ]; then
                    sudo mkdir -p "$COOKIES_DIR"
                fi
                # Create empty file if it doesn't exist
                if [ ! -f "$COOKIES_FILE" ]; then
                    sudo touch "$COOKIES_FILE"
                fi
                # Set permissions: readable and writable by botuser
                sudo chown botuser:botuser "$COOKIES_FILE"
                sudo chmod 644 "$COOKIES_FILE"
            fi
        done
        echo "Cookies files configured with proper permissions"
    fi
else
    echo "Warning: .env file not found in $PROJECT_DIR"
fi

# Install telegram-bot-api if not installed
if [ ! -f "/usr/local/bin/telegram-bot-api" ]; then
    echo "telegram-bot-api not found. Please install it first:"
    echo "  wget https://github.com/tdlib/telegram-bot-api/releases/download/v7.0.0/telegram-bot-api_Linux_arm64 -O /tmp/telegram-bot-api"
    echo "  sudo mv /tmp/telegram-bot-api /usr/local/bin/telegram-bot-api"
    echo "  sudo chmod +x /usr/local/bin/telegram-bot-api"
    echo ""
    if [ -t 0 ]; then
        # Interactive mode
        read -p "Do you want to download and install telegram-bot-api now? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            echo "Downloading telegram-bot-api..."
            wget -q https://github.com/tdlib/telegram-bot-api/releases/download/v7.0.0/telegram-bot-api_Linux_arm64 -O /tmp/telegram-bot-api
            sudo mv /tmp/telegram-bot-api /usr/local/bin/telegram-bot-api
            sudo chmod +x /usr/local/bin/telegram-bot-api
            sudo chown botuser:botuser /usr/local/bin/telegram-bot-api
            echo "telegram-bot-api installed successfully"
        else
            echo "Skipping telegram-bot-api installation. Please install it manually."
        fi
    else
        # Non-interactive mode - skip installation
        echo "Non-interactive mode detected. Skipping telegram-bot-api installation."
        echo "Please install it manually using the commands shown above."
    fi
fi

# Install systemd services
sudo cp "$SCRIPT_DIR/telegram-bot-api.service" /etc/systemd/system/
sudo cp "$SCRIPT_DIR/envedour-down-bot.service" /etc/systemd/system/
sudo systemctl daemon-reload
echo "Systemd services installed"

# Install sysctl optimizations
if [ -f "$SCRIPT_DIR/sysctl.conf" ]; then
    sudo cp "$SCRIPT_DIR/sysctl.conf" /etc/sysctl.d/99-arm-optimizations.conf
    sudo sysctl -p /etc/sysctl.d/99-arm-optimizations.conf
fi

# Install limits
if [ -f "$SCRIPT_DIR/limits.conf" ]; then
    sudo cp "$SCRIPT_DIR/limits.conf" /etc/security/limits.d/99-botuser.conf
fi

# Setup tmpfs
if ! grep -q "/dev/shm/videos" /etc/fstab; then
    echo "tmpfs /dev/shm/videos tmpfs size=2G,nr_blocks=524288,nr_inodes=65536,noatime,nodev,nosuid 0 0" | sudo tee -a /etc/fstab
    sudo mkdir -p /dev/shm/videos
    sudo mount /dev/shm/videos
    sudo chmod 1777 /dev/shm/videos
fi

echo "Installation complete!"
echo "Don't forget to:"
echo "1. Set TELEGRAM_API_ID and TELEGRAM_API_HASH in /opt/telegram-bot-api/.env"
echo "2. Set BOT_TOKEN and other variables in /opt/envedour-bot/.env"
echo "3. Configure Redis if needed"
echo "4. Run: sudo systemctl enable telegram-bot-api envedour-down-bot"
echo "5. Run: sudo systemctl start telegram-bot-api envedour-down-bot"
