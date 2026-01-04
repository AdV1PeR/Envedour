# Changelog

## [1.0.0] - Initial Release

### Added
- Complete Telegram bot implementation with ARM64 optimizations
- Redis-based job queue with priority support
- yt-dlp + aria2c + FFmpeg integration
- Thermal monitoring and throttling for ARM devices
- Memory management with ARM-specific limits
- Systemd service configuration
- Docker support with multi-stage builds
- Comprehensive deployment scripts
- Monitoring and backup scripts
- Network tuning for 100Mbps Ethernet
- Support for donor users with high priority queue

### Features
- ARM NEON SIMD acceleration support
- tmpfs-based temporary storage (2GB)
- Concurrent processing (4 workers + 2 spare)
- File size limit: 2GB
- Automatic thermal throttling at 85°C
- Memory reservation: 512MB for system
- Redis persistence disabled for RAM-only operation
- Local Telegram Bot API support

### Optimizations
- ARM64-specific build flags
- Goroutine pool for efficient concurrency
- sync.Pool for message object reuse
- ARM-optimized Redis configuration
- CPU affinity and thermal management
- Network interface tuning
