#!/bin/bash
# Script to build FFmpeg with ARM NEON optimizations
# This is optional if system FFmpeg doesn't have NEON support

set -e

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_DIR="$SCRIPT_DIR/../build"

echo "Building FFmpeg with ARM NEON optimizations..."
echo "Build directory: $BUILD_DIR"

# Create build directory
mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

# Install build dependencies
sudo apt-get update
sudo apt-get install -y \
    build-essential \
    yasm \
    nasm \
    libx264-dev \
    libx265-dev \
    libvpx-dev \
    libfdk-aac-dev \
    libmp3lame-dev \
    libopus-dev

# Clone FFmpeg
if [ ! -d "ffmpeg" ]; then
    git clone https://git.ffmpeg.org/ffmpeg.git
fi

cd "$BUILD_DIR/ffmpeg"
git checkout release/6.1

# Configure with ARM optimizations
./configure \
    --arch=arm64 \
    --enable-neon \
    --enable-vfp \
    --enable-hardcoded-tables \
    --enable-gpl \
    --enable-libx264 \
    --enable-libx265 \
    --enable-libvpx \
    --enable-libfdk-aac \
    --enable-libmp3lame \
    --enable-libopus \
    --enable-shared \
    --disable-static

# Build
make -j4
sudo make install
sudo ldconfig

echo "FFmpeg with ARM NEON built and installed"
