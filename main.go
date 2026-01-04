package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"envedour-bot/internal/bot"
	"envedour-bot/internal/config"
	"envedour-bot/internal/executor"
	"envedour-bot/internal/queue"
	"envedour-bot/internal/thermal"
)

func main() {
	armOptimized := flag.Bool("arm-optimized", false, "Enable ARM-specific optimizations")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Ошибка конфигурации:\n%v\n\nПроверьте настройки и попробуйте снова.", err)
	}

	// Initialize thermal monitoring if ARM optimized
	if *armOptimized {
		thermalMonitor := thermal.NewMonitor()
		go thermalMonitor.Start()
		defer thermalMonitor.Stop()
	}

	// Initialize Redis queue
	log.Printf("Connecting to Redis at %s...", cfg.RedisAddr)
	redisQueue, err := queue.NewRedisQueue(cfg.RedisAddr)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisQueue.Close()
	log.Printf("✓ Redis connected")

	// Initialize executor
	exec := executor.NewExecutor(cfg, *armOptimized)
	defer exec.Close()

	// Initialize bot
	apiURL := "Telegram API"
	if cfg.LocalAPIURL != "" {
		apiURL = cfg.LocalAPIURL
	}
	log.Printf("Connecting to %s...", apiURL)
	envedourBot, err := bot.NewBot(cfg, redisQueue, exec)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}
	log.Printf("✓ Bot initialized")

	// Start worker pool
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start executor workers
	for i := 0; i < cfg.WorkerCount; i++ {
		go exec.Worker(ctx, redisQueue)
	}

	// Start bot
	go envedourBot.Start(ctx)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	cancel()
	time.Sleep(2 * time.Second)
}
