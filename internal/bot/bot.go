package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"envedour-bot/internal/config"
	"envedour-bot/internal/executor"
	"envedour-bot/internal/queue"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api         *tgbotapi.BotAPI
	config      *config.Config
	queue       queue.Queue
	executor    *executor.Executor
	workerPool  chan struct{}
	preferences *PreferencesStore
}

func NewBot(cfg *config.Config, q queue.Queue, exec *executor.Executor) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, err
	}

	// Use local API if configured
	// SetAPIEndpoint expects format with placeholders: "http://localhost:8081/bot%s/%s"
	// We need to escape % to %% so fmt.Sprintf in library works correctly
	if cfg.LocalAPIURL != "" {
		// Remove trailing slash if present
		baseURL := strings.TrimSuffix(cfg.LocalAPIURL, "/")
		// Add http:// if protocol is missing
		if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
			baseURL = "http://" + baseURL
		}
		// Escape % signs for proper formatting: %s becomes %%s
		endpoint := fmt.Sprintf("%s/bot%%s/%%s", baseURL)
		api.SetAPIEndpoint(endpoint)
	}

	// Initialize preferences store
	var prefsStore *PreferencesStore
	if redisClient := q.GetClient(); redisClient != nil {
		prefsStore = NewPreferencesStore(redisClient)
	}

	bot := &Bot{
		api:         api,
		config:      cfg,
		queue:       q,
		executor:    exec,
		preferences: prefsStore,
		workerPool:  make(chan struct{}, cfg.WorkerCount+2),
	}

	return bot, nil
}

func (b *Bot) Start(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30 // 30s timeout for poor connections

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			return
		case update := <-updates:
			b.workerPool <- struct{}{} // Acquire worker
			go b.handleUpdate(update)
		}
	}
}

func (b *Bot) handleUpdate(update tgbotapi.Update) {
	defer func() { <-b.workerPool }() // Release worker

	// Handle callback queries (button presses)
	if update.CallbackQuery != nil {
		b.handleCallbackQuery(update.CallbackQuery)
		return
	}

	if update.Message == nil {
		return
	}

	msg := update.Message
	chatID := msg.Chat.ID

	// Check if user is donor
	isDonor := b.isDonor(chatID)

	switch {
	case msg.IsCommand():
		b.handleCommand(msg, isDonor)
	case msg.Text != "":
		b.handleURL(msg, isDonor)
	}
}

func (b *Bot) handleCommand(msg *tgbotapi.Message, isDonor bool) {
	chatID := msg.Chat.ID
	command := msg.Command()

	switch command {
	case "start":
		helpText := "Привет! Я бот для скачивания видео.\n\n" +
			"📥 Отправь ссылку на видео для скачивания\n\n" +
			"Используй кнопки ниже для настройки:"
		msg := tgbotapi.NewMessage(chatID, helpText)
		msg.ReplyMarkup = createMainKeyboard()
		b.api.Send(msg)
		return
	case "help":
		helpText := "📋 Справка:\n\n" +
			"💡 Просто отправь ссылку на видео для скачивания!\n\n" +
			"Используй кнопки для настройки качества и типа медиа.\n\n" +
			"⚙️ Качество - выбери разрешение видео\n" +
			"🎵 Аудио/Видео - выбери тип скачивания\n" +
			"📊 Статус - посмотри очередь и настройки"
		msg := tgbotapi.NewMessage(chatID, helpText)
		msg.ReplyMarkup = createMainKeyboard()
		b.api.Send(msg)
		return
	case "status":
		b.showStatus(chatID)
	case "quality", "audio", "video":
		// These commands are now handled via inline buttons
		// Show main menu
		msg := tgbotapi.NewMessage(chatID, "Используй кнопки для настройки:")
		msg.ReplyMarkup = createMainKeyboard()
		b.api.Send(msg)
	default:
		b.sendMessage(chatID, "❌ Неизвестная команда. Используйте /help")
	}
}

func (b *Bot) handleURL(msg *tgbotapi.Message, isDonor bool) {
	url := msg.Text
	chatID := msg.Chat.ID

	// Validate URL - check if it's a valid URL format
	// Also check for URL entities in case Telegram parsed it
	if !isValidURL(url) {
		// Check if message has URL entities
		if len(msg.Entities) > 0 {
			for _, entity := range msg.Entities {
				if entity.Type == "url" {
					url = msg.Text[entity.Offset : entity.Offset+entity.Length]
					break
				}
			}
		}
		// If still not valid, reject
		if !isValidURL(url) {
			b.sendMessage(chatID, "Пожалуйста, отправьте валидную ссылку на видео.")
			return
		}
	}

	// Check if URL is from Instagram or TikTok - auto-download best quality
	if isInstagramURL(url) || isTikTokURL(url) {
		// For Instagram/TikTok always use video (not audio)
		// Create job with best quality automatically
		job := &queue.Job{
			ID:        generateJobID(),
			URL:       url,
			ChatID:    chatID,
			Priority:  queue.PriorityLow,
			Quality:   "best",
			MediaType: "video", // Always video for Instagram/TikTok
			CreatedAt: time.Now(),
		}

		if isDonor {
			job.Priority = queue.PriorityHigh
		}

		// Add to queue
		if err := b.queue.Enqueue(job); err != nil {
			b.sendMessage(chatID, "Ошибка при добавлении задачи в очередь. Попробуйте позже.")
			return
		}

		return
	}

	// For other platforms (YouTube, etc.) - show quality selection keyboard
	// Generate a temporary job ID for this URL
	jobID := generateJobID()

	// Save URL temporarily in Redis (will be retrieved when user selects quality)
	if b.preferences != nil {
		if err := b.preferences.SavePendingURL(jobID, url); err != nil {
			b.sendMessage(chatID, "❌ Ошибка при обработке ссылки. Попробуйте позже.")
			return
		}
	}

	// Show quality selection keyboard
	text := "📥 Выбери качество для скачивания:"
	keyboard := createDownloadQualityKeyboard(jobID)
	message := tgbotapi.NewMessage(chatID, text)
	message.ReplyMarkup = &keyboard
	b.api.Send(message)
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	b.api.Send(msg)
}

func (b *Bot) isDonor(chatID int64) bool {
	for _, id := range b.config.DonorChatIDs {
		if id == chatID {
			return true
		}
	}
	return false
}

func isValidURL(s string) bool {
	if len(s) < 8 {
		return false
	}
	if len(s) >= 7 && s[:7] == "http://" {
		return true
	}
	if len(s) >= 8 && s[:8] == "https://" {
		return true
	}
	return false
}

func isInstagramURL(url string) bool {
	return strings.Contains(url, "instagram.com") || strings.Contains(url, "instagr.am")
}

func isTikTokURL(url string) bool {
	return strings.Contains(url, "tiktok.com") || strings.Contains(url, "vt.tiktok.com")
}

func generateJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}

func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	chatID := query.Message.Chat.ID
	data := query.Data

	// Answer callback to remove loading state
	callback := tgbotapi.NewCallback(query.ID, "")
	b.api.Request(callback)

	switch {
	case data == "menu_main":
		keyboard := createMainKeyboard()
		msg := tgbotapi.NewEditMessageText(chatID, query.Message.MessageID, "Главное меню:")
		msg.ReplyMarkup = &keyboard
		b.api.Send(msg)

	case data == "menu_quality":
		var prefs *UserPreferences
		if b.preferences != nil {
			prefs = b.preferences.GetPreferences(chatID)
		}
		if prefs == nil {
			prefs = &UserPreferences{Quality: "best", MediaType: "video"}
		}
		text := fmt.Sprintf("⚙️ Выбери качество:\n\nТекущее: %s", prefs.Quality)
		keyboard := createQualityKeyboard()
		msg := tgbotapi.NewEditMessageText(chatID, query.Message.MessageID, text)
		msg.ReplyMarkup = &keyboard
		b.api.Send(msg)

	case data == "menu_media":
		var prefs *UserPreferences
		if b.preferences != nil {
			prefs = b.preferences.GetPreferences(chatID)
		}
		if prefs == nil {
			prefs = &UserPreferences{Quality: "best", MediaType: "video"}
		}
		text := fmt.Sprintf("🎵 Выбери тип:\n\nТекущий: %s", prefs.MediaType)
		keyboard := createMediaTypeKeyboard()
		msg := tgbotapi.NewEditMessageText(chatID, query.Message.MessageID, text)
		msg.ReplyMarkup = &keyboard
		b.api.Send(msg)

	case data == "cmd_status":
		b.showStatus(chatID)
		// Delete the button message
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, query.Message.MessageID)
		b.api.Send(deleteMsg)

	case strings.HasPrefix(data, "quality_"):
		if b.preferences == nil {
			b.sendMessage(chatID, "❌ Система предпочтений недоступна")
			return
		}
		quality := strings.TrimPrefix(data, "quality_")
		if err := b.preferences.SetQuality(chatID, quality); err != nil {
			b.sendMessage(chatID, "❌ Ошибка при сохранении настроек")
			return
		}
		mediaType := "видео"
		if quality == "audio" {
			mediaType = "аудио"
		}
		text := fmt.Sprintf("✅ Качество установлено: %s (%s)", quality, mediaType)
		keyboard := createQualityKeyboard()
		msg := tgbotapi.NewEditMessageText(chatID, query.Message.MessageID, text)
		msg.ReplyMarkup = &keyboard
		b.api.Send(msg)

	case strings.HasPrefix(data, "media_"):
		if b.preferences == nil {
			b.sendMessage(chatID, "❌ Система предпочтений недоступна")
			return
		}
		mediaType := strings.TrimPrefix(data, "media_")
		if err := b.preferences.SetMediaType(chatID, mediaType); err != nil {
			b.sendMessage(chatID, "❌ Ошибка при сохранении настроек")
			return
		}
		var text string
		if mediaType == "audio" {
			text = "✅ Режим установлен: скачивание аудио (MP3)\n\nТеперь отправь ссылку на видео."
		} else {
			prefs := b.preferences.GetPreferences(chatID)
			text = fmt.Sprintf("✅ Режим установлен: скачивание видео\nКачество: %s", prefs.Quality)
		}
		keyboard := createMediaTypeKeyboard()
		msg := tgbotapi.NewEditMessageText(chatID, query.Message.MessageID, text)
		msg.ReplyMarkup = &keyboard
		b.api.Send(msg)

	case strings.HasPrefix(data, "dl_q_"):
		// Format: dl_q_<quality>:<jobID>
		parts := strings.SplitN(data, ":", 2)
		if len(parts) != 2 {
			b.sendMessage(chatID, "❌ Ошибка: неверный формат запроса")
			return
		}
		qualityPart := strings.TrimPrefix(parts[0], "dl_q_")
		jobID := parts[1]

		// Retrieve URL from Redis
		if b.preferences == nil {
			b.sendMessage(chatID, "❌ Система недоступна. Попробуйте отправить ссылку снова.")
			return
		}

		url, err := b.preferences.GetPendingURL(jobID)
		if err != nil {
			b.sendMessage(chatID, "❌ Ссылка устарела или не найдена. Пожалуйста, отправьте ссылку снова.")
			return
		}

		// Validate URL
		if !isValidURL(url) {
			b.sendMessage(chatID, "❌ Неверная ссылка")
			return
		}

		// Determine media type based on quality
		mediaType := "video"
		if qualityPart == "audio" {
			mediaType = "audio"
		}

		// Save preferences if available
		if b.preferences != nil {
			if qualityPart == "audio" {
				b.preferences.SetMediaType(chatID, "audio")
			} else {
				b.preferences.SetQuality(chatID, qualityPart)
			}
		}

		// Create job
		job := &queue.Job{
			ID:        generateJobID(),
			URL:       url,
			ChatID:    chatID,
			Priority:  queue.PriorityLow,
			Quality:   qualityPart,
			MediaType: mediaType,
			CreatedAt: time.Now(),
		}

		if b.isDonor(chatID) {
			job.Priority = queue.PriorityHigh
		}

		// Add to queue
		if err := b.queue.Enqueue(job); err != nil {
			b.sendMessage(chatID, "❌ Ошибка при добавлении задачи в очередь. Попробуйте позже.")
			return
		}

		// Delete the keyboard message
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, query.Message.MessageID)
		b.api.Send(deleteMsg)
	}
}

func (b *Bot) showStatus(chatID int64) {
	status := b.queue.GetStatus()
	var prefs *UserPreferences
	if b.preferences != nil {
		prefs = b.preferences.GetPreferences(chatID)
	}
	if prefs == nil {
		prefs = &UserPreferences{Quality: "best", MediaType: "video"}
	}
	text := fmt.Sprintf("📊 Очередь: %d задач\n\n⚙️ Текущие настройки:\nКачество: %s\nТип: %s", status, prefs.Quality, prefs.MediaType)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = createMainKeyboard()
	b.api.Send(msg)
}
