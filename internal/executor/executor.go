package executor

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"envedour-bot/internal/config"
	"envedour-bot/internal/queue"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ThermalMonitor interface for thermal monitoring (only on ARM64)
type ThermalMonitor interface {
	IsThrottled() bool
	Start()
	Stop()
}

var thermalMonitorImpl ThermalMonitor

type Executor struct {
	config       *config.Config
	armOptimized bool
	botAPI       *tgbotapi.BotAPI
	thermalMon   ThermalMonitor
}

func NewExecutor(cfg *config.Config, armOptimized bool) *Executor {
	// Initialize bot API for sending files
	botAPI, _ := tgbotapi.NewBotAPI(cfg.BotToken)
	if botAPI != nil && cfg.LocalAPIURL != "" {
		// Remove trailing slash if present
		baseURL := strings.TrimSuffix(cfg.LocalAPIURL, "/")
		// Add http:// if protocol is missing
		if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
			baseURL = "http://" + baseURL
		}
		// Escape % signs for proper formatting: %s becomes %%s
		endpoint := fmt.Sprintf("%s/bot%%s/%%s", baseURL)
		botAPI.SetAPIEndpoint(endpoint)
	}

	exec := &Executor{
		config:       cfg,
		armOptimized: armOptimized,
		botAPI:       botAPI,
	}

	// Initialize thermal monitor on ARM64 if requested
	if armOptimized && runtime.GOARCH == "arm64" && thermalMonitorImpl != nil {
		exec.thermalMon = thermalMonitorImpl
		go func() {
			if m, ok := exec.thermalMon.(interface{ Start() }); ok {
				m.Start()
			}
		}()
	}

	// Validate environment (errors ignored)
	ValidateEnvironment(cfg.TmpfsPath)

	return exec
}

func (e *Executor) Worker(ctx context.Context, q queue.Queue) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			job, err := q.Dequeue(ctx)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}

			if job == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			e.processJob(ctx, job)
		}
	}
}

func (e *Executor) processJob(ctx context.Context, job *queue.Job) {
	// Check thermal throttling if ARM optimized
	if e.armOptimized && e.thermalMon != nil {
		if e.thermalMon.IsThrottled() {
			e.sendMessage(job.ChatID, "Система перегружена (высокая температура). Попробуйте позже.")
			time.Sleep(5 * time.Second)
			return
		}
	}

	// Check available memory
	if err := e.checkMemory(); err != nil {
		e.sendMessage(job.ChatID, "Недостаточно памяти на устройстве. Попробуйте позже.")
		return
	}

	// Set default values if not set
	quality := job.Quality
	if quality == "" {
		quality = "best"
	}
	mediaType := job.MediaType
	if mediaType == "" {
		mediaType = "video"
	}

	// Download media (video or audio)
	filePath, err := e.downloadMedia(ctx, job.URL, job.ID, quality, mediaType)
	if err != nil {
		log.Printf("Download error: %v", err)
		userMsg := "❌ Ошибка при скачивании.\n\nВозможные причины:\n• Неверная ссылка\n• Видео недоступно\n• Проблемы с сетью\n• Недостаточно памяти\n\nПопробуйте другую ссылку или повторите позже."
		e.sendMessage(job.ChatID, userMsg)
		return
	}
	defer os.Remove(filePath)

	// Send media based on type
	if mediaType == "audio" {
		if err := e.sendAudio(job.ChatID, filePath); err != nil {
			log.Printf("Audio send error: %v", err)
			userMsg := "❌ Ошибка при отправке аудио.\n\nВозможно, файл слишком большой или поврежден.\nПопробуйте другую ссылку."
			e.sendMessage(job.ChatID, userMsg)
			return
		}
	} else {
		if err := e.sendVideo(job.ChatID, filePath); err != nil {
			log.Printf("Video send error: %v", err)
			userMsg := "❌ Ошибка при отправке видео.\n\n"
			if err.Error() == "bot API not initialized" {
				userMsg += "Проблема с конфигурацией бота. Обратитесь к администратору."
			} else if filepath.Ext(filePath) != "" {
				userMsg += "Возможно, файл слишком большой или поврежден.\nПопробуйте другую ссылку."
			} else {
				userMsg += "Попробуйте повторить запрос позже."
			}
			e.sendMessage(job.ChatID, userMsg)
			return
		}
	}
}

func (e *Executor) downloadMedia(ctx context.Context, url, jobID, quality, mediaType string) (string, error) {
	var outputPath string
	var title string
	var tempCookiesFiles []string // Track temp cookies files for cleanup

	// Helper function to create temp cookies file
	createTempCookies := func(originalFile string) string {
		tempCookiesFile := filepath.Join(e.config.TmpfsPath, fmt.Sprintf("cookies_%s_%d.txt", jobID, time.Now().UnixNano()))
		if data, err := os.ReadFile(originalFile); err == nil {
			if err := os.WriteFile(tempCookiesFile, data, 0644); err == nil {
				os.Chmod(tempCookiesFile, 0644)
				tempCookiesFiles = append(tempCookiesFiles, tempCookiesFile)
				return tempCookiesFile
			}
		}
		return originalFile // Fallback to original
	}

	// Cleanup temp cookies files at the end
	defer func() {
		for _, tempFile := range tempCookiesFiles {
			os.Remove(tempFile)
		}
	}()

	// For audio, get title first to use in filename (to avoid truncation)
	if mediaType == "audio" {
		// Get video title using yt-dlp --print (before downloading)
		titleArgs := []string{"--no-cache-dir", "--print", "%(title)s", url}
		if cookiesFile := e.getCookiesFile(url); cookiesFile != "" {
			tempCookies := createTempCookies(cookiesFile)
			titleArgs = append(titleArgs, "--cookies", tempCookies)
		}
		// Add TikTok-specific options for title extraction
		if strings.Contains(url, "tiktok.com") || strings.Contains(url, "vt.tiktok.com") {
			titleArgs = append(titleArgs,
				"--no-check-certificate",
				"-4", // Force IPv4
				"--legacy-server-connect",
				"--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
				"--referer", "https://www.tiktok.com/",
				"--add-header", "Accept:text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
				"--add-header", "Accept-Language:en-US,en;q=0.9",
				"--add-header", "Accept-Encoding:gzip, deflate, br",
				"--add-header", "DNT:1",
				"--add-header", "Connection:keep-alive",
				"--add-header", "Upgrade-Insecure-Requests:1",
				"--add-header", "Sec-Fetch-Dest:document",
				"--add-header", "Sec-Fetch-Mode:navigate",
				"--add-header", "Sec-Fetch-Site:none",
				"--add-header", "Sec-Fetch-User:?1",
			)
		}
		
		titleCmd := exec.CommandContext(ctx, "yt-dlp", titleArgs...)
		titleBytes, err := titleCmd.Output()
		if err == nil && len(titleBytes) > 0 {
			title = strings.TrimSpace(string(titleBytes))
			// Sanitize title for filesystem
			title = sanitizeFilename(title)
			// Limit to reasonable length (220 chars) to avoid filesystem limits
			if len(title) > 220 {
				title = title[:220]
			}
		}
		// If title extraction failed or empty, use timestamp-based name
		if title == "" {
			title = fmt.Sprintf("audio_%d", time.Now().Unix())
		}
		// Use only title in output path (no jobID prefix)
		outputPath = filepath.Join(e.config.TmpfsPath, fmt.Sprintf("%s.%%(ext)s", title))
	} else {
		outputPath = filepath.Join(e.config.TmpfsPath, fmt.Sprintf("%s.%%(ext)s", jobID))
	}

	// Build yt-dlp command arguments
	args := []string{
		"--no-cache-dir",
		"--external-downloader", "aria2c",
		"--external-downloader-args", "aria2c:-j16 -x16 -s16 --file-allocation=falloc --stream-piece-selector=geom --max-download-limit=0",
		"-o", outputPath,
		// Disable automatic cookie extraction from browsers (server doesn't have browsers)
		"--no-cookies-from-browser",
	}

	// Add TikTok-specific options to bypass restrictions
	if strings.Contains(url, "tiktok.com") || strings.Contains(url, "vt.tiktok.com") {
		// TikTok requires specific headers and settings
		args = append(args,
			"--no-check-certificate",
			"-4", // Force IPv4
			"--legacy-server-connect",
			"--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
			"--referer", "https://www.tiktok.com/",
			"--add-header", "Accept:text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
			"--add-header", "Accept-Language:en-US,en;q=0.9",
			"--add-header", "Accept-Encoding:gzip, deflate, br",
			"--add-header", "DNT:1",
			"--add-header", "Connection:keep-alive",
			"--add-header", "Upgrade-Insecure-Requests:1",
			"--add-header", "Sec-Fetch-Dest:document",
			"--add-header", "Sec-Fetch-Mode:navigate",
			"--add-header", "Sec-Fetch-Site:none",
			"--add-header", "Sec-Fetch-User:?1",
			"--add-header", "sec-ch-ua:\"Google Chrome\";v=\"131\", \"Chromium\";v=\"131\", \"Not_A Brand\";v=\"24\"",
			"--add-header", "sec-ch-ua-mobile:?0",
			"--add-header", "sec-ch-ua-platform:\"Windows\"",
		)
	} else {
		// For other sites, use standard SSL settings
		args = append(args, "--legacy-server-connect")
	}

	// Set format based on media type and quality
	if mediaType == "audio" {
		args = append(args, "-x", "--audio-format", "mp3", "--audio-quality", "0")
	} else {
		format := e.getFormatForQuality(quality)
		args = append(args, "--format", format)
	}

	// Add cookies file if configured (platform-specific or fallback)
	// Use temporary copy for cookies to avoid permission issues when yt-dlp tries to save them
	if cookiesFile := e.getCookiesFile(url); cookiesFile != "" {
		tempCookies := createTempCookies(cookiesFile)
		args = append(args, "--cookies", tempCookies)
	} else if strings.Contains(url, "tiktok.com") || strings.Contains(url, "vt.tiktok.com") {
		log.Printf("Warning: TikTok URL detected but no cookies file configured. TikTok may require cookies to bypass 403 errors.")
	}

	// Add URL at the end
	args = append(args, url)

	// Use yt-dlp to get video info and download
	ytdlpCmd := exec.CommandContext(ctx, "yt-dlp", args...)

	ytdlpCmd.Env = os.Environ()
	if e.armOptimized && runtime.GOARCH == "arm64" {
		// Set ARM-specific environment
		ytdlpCmd.Env = append(ytdlpCmd.Env, "FFMPEG_BINARY=ffmpeg")
	}

	output, err := ytdlpCmd.CombinedOutput()
	// Clean up temp cookies files even if download fails
	for _, tempFile := range tempCookiesFiles {
		os.Remove(tempFile)
	}
	if err != nil {
		return "", fmt.Errorf("yt-dlp failed: %w\nOutput: %s", err, output)
	}

	// Find the downloaded file
	var matches []string
	if mediaType == "audio" {
		// For audio, look for MP3 file with title name
		if title != "" && !strings.HasPrefix(title, "audio_") {
			// Try exact match first
			pattern := filepath.Join(e.config.TmpfsPath, title+".mp3")
			if _, err := os.Stat(pattern); err == nil {
				matches = []string{pattern}
			}
		}
		// If not found, find most recently created MP3 file
		if len(matches) == 0 {
			pattern := filepath.Join(e.config.TmpfsPath, "*.mp3")
			allMatches, _ := filepath.Glob(pattern)
			if len(allMatches) > 0 {
				// Sort by modification time, get most recent
				var mostRecent string
				var mostRecentTime time.Time
				for _, match := range allMatches {
					if stat, err := os.Stat(match); err == nil {
						if stat.ModTime().After(mostRecentTime) {
							mostRecentTime = stat.ModTime()
							mostRecent = match
						}
					}
				}
				if mostRecent != "" {
					matches = []string{mostRecent}
					// If title was not extracted before, try to extract it now and rename file
					if title == "" || strings.HasPrefix(title, "audio_") {
						titleArgs := []string{"--no-cache-dir", "--print", "%(title)s", url}
						if cookiesFile := e.getCookiesFile(url); cookiesFile != "" {
							tempCookies := createTempCookies(cookiesFile)
							titleArgs = append(titleArgs, "--cookies", tempCookies)
						}
						if strings.Contains(url, "tiktok.com") || strings.Contains(url, "vt.tiktok.com") {
							titleArgs = append(titleArgs,
								"--no-check-certificate",
								"-4", // Force IPv4
								"--legacy-server-connect",
								"--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
								"--referer", "https://www.tiktok.com/",
								"--add-header", "Accept:text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
								"--add-header", "Accept-Language:en-US,en;q=0.9",
								"--add-header", "Accept-Encoding:gzip, deflate, br",
								"--add-header", "DNT:1",
								"--add-header", "Connection:keep-alive",
								"--add-header", "Upgrade-Insecure-Requests:1",
								"--add-header", "Sec-Fetch-Dest:document",
								"--add-header", "Sec-Fetch-Mode:navigate",
								"--add-header", "Sec-Fetch-Site:none",
								"--add-header", "Sec-Fetch-User:?1",
							)
						}
						titleCmd := exec.CommandContext(ctx, "yt-dlp", titleArgs...)
						if titleBytes, err := titleCmd.Output(); err == nil && len(titleBytes) > 0 {
							newTitle := strings.TrimSpace(string(titleBytes))
							newTitle = sanitizeFilename(newTitle)
							if len(newTitle) > 220 {
								newTitle = newTitle[:220]
							}
							if newTitle != "" && !strings.HasPrefix(newTitle, "audio_") {
								// Rename file to use proper title
								newPath := filepath.Join(e.config.TmpfsPath, newTitle+".mp3")
								if err := os.Rename(mostRecent, newPath); err == nil {
									matches = []string{newPath}
								}
							}
						}
					}
				}
			}
		}
	} else {
		// For video, look for any file with jobID prefix
		pattern := filepath.Join(e.config.TmpfsPath, jobID+"*")
		matches, _ = filepath.Glob(pattern)
	}
	
	if len(matches) == 0 {
		return "", fmt.Errorf("downloaded file not found")
	}

	return matches[0], nil
}

// getCookiesFile returns the appropriate cookies file path for the given URL
// Returns empty string if no cookies file is configured for the platform
func (e *Executor) getCookiesFile(url string) string {
	// Check for platform-specific cookies files first
	if strings.Contains(url, "tiktok.com") || strings.Contains(url, "vt.tiktok.com") {
		if e.config.TikTokCookies != "" {
			if _, err := os.Stat(e.config.TikTokCookies); err == nil {
				return e.config.TikTokCookies
			}
		}
	} else if strings.Contains(url, "instagram.com") || strings.Contains(url, "instagr.am") {
		if e.config.InstagramCookies != "" {
			if _, err := os.Stat(e.config.InstagramCookies); err == nil {
				return e.config.InstagramCookies
			}
		}
	} else if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") {
		if e.config.YouTubeCookies != "" {
			if _, err := os.Stat(e.config.YouTubeCookies); err == nil {
				return e.config.YouTubeCookies
			}
		}
	}
	
	// Fallback to deprecated COOKIES_FILE for backward compatibility
	if e.config.CookiesFile != "" {
		if _, err := os.Stat(e.config.CookiesFile); err == nil {
			return e.config.CookiesFile
		}
	}
	
	return ""
}

// sanitizeFilename removes or replaces characters that are invalid in filenames
func sanitizeFilename(name string) string {
	// Replace invalid filesystem characters
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "\x00"}
	result := name
	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}
	// Remove leading/trailing spaces and dots
	result = strings.Trim(result, " .")
	// Replace multiple spaces/underscores with single underscore
	result = strings.ReplaceAll(result, "  ", " ")
	result = strings.ReplaceAll(result, "__", "_")
	return result
}

func (e *Executor) getFormatForQuality(quality string) string {
	switch quality {
	case "1080p":
		return "bestvideo[height<=1080][ext=mp4]+bestaudio[ext=m4a]/best[height<=1080][ext=mp4]/best"
	case "720p":
		return "bestvideo[height<=720][ext=mp4]+bestaudio[ext=m4a]/best[height<=720][ext=mp4]/best"
	case "480p":
		return "bestvideo[height<=480][ext=mp4]+bestaudio[ext=m4a]/best[height<=480][ext=mp4]/best"
	case "360p":
		return "bestvideo[height<=360][ext=mp4]+bestaudio[ext=m4a]/best[height<=360][ext=mp4]/best"
	case "audio":
		return "bestaudio[ext=m4a]/bestaudio/best"
	default: // "best"
		return "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best"
	}
}

func (e *Executor) sendAudio(chatID int64, audioPath string) error {
	if e.botAPI == nil {
		return fmt.Errorf("bot API not initialized")
	}

	file, err := os.Open(audioPath)
	if err != nil {
		return fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat audio file: %w", err)
	}

	// Check file size
	if stat.Size() > e.config.MaxFileSize {
		return fmt.Errorf("file too large: %d bytes (max: %d)", stat.Size(), e.config.MaxFileSize)
	}

	audio := tgbotapi.NewAudio(chatID, tgbotapi.FilePath(audioPath))

	_, err = e.botAPI.Send(audio)
	if err != nil {
		return fmt.Errorf("failed to send audio: %w", err)
	}
	return nil
}

func (e *Executor) sendVideo(chatID int64, videoPath string) error {
	if e.botAPI == nil {
		return fmt.Errorf("bot API not initialized")
	}

	file, err := os.Open(videoPath)
	if err != nil {
		return fmt.Errorf("failed to open video file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat video file: %w", err)
	}

	// Check file size
	if stat.Size() > e.config.MaxFileSize {
		return fmt.Errorf("file too large: %d bytes (max: %d)", stat.Size(), e.config.MaxFileSize)
	}

	video := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(videoPath))
	video.SupportsStreaming = true

	_, err = e.botAPI.Send(video)
	if err != nil {
		return fmt.Errorf("failed to send video: %w", err)
	}
	return nil
}

func (e *Executor) sendMessage(chatID int64, text string) {
	if e.botAPI == nil {
		return
	}
	msg := tgbotapi.NewMessage(chatID, text)
	e.botAPI.Send(msg)
}

func (e *Executor) checkMemory() error {
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		return err
	}

	// Use configurable minimum free memory (default 256MB)
	minFreeMemory := uint64(e.config.MinFreeMemMB * 1024 * 1024)
	freeMB := info.Freeram / (1024 * 1024)

	if info.Freeram < minFreeMemory {
		log.Printf("Memory check failed: %d MB free (minimum: %d MB)", freeMB, e.config.MinFreeMemMB)
		return fmt.Errorf("insufficient memory: %d MB free (minimum: %d MB)", freeMB, e.config.MinFreeMemMB)
	}

	return nil
}

func (e *Executor) Close() {
	if e.thermalMon != nil {
		if m, ok := e.thermalMon.(interface{ Stop() }); ok {
			m.Stop()
		}
	}
}
