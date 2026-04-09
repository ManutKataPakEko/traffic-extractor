package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("failed to load .env: %v", err)
	}

	cfg := loadConfig()
	app, err := newApp(cfg)
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}
	defer app.close()

	r := fiber.New()
	app.registerRoutes(r)

	if err := r.Listen(cfg.ListenAddr); err != nil {
		log.Fatalf("fiber listen error: %v", err)
	}
}

func loadConfig() Config {
	return Config{
		ListenAddr:        defaultListenAddr,
		LogPath:           defaultLogPath,
		MLRPCAddr:         getenvOrDefault("ML_RPC_ADDR", defaultMLRPCAddr),
		MLRPCTimeout:      defaultMLRPCTimeout,
		MaxBodyBytes:      defaultMaxBodyBytes,
		TelegramBotToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:    os.Getenv("TELEGRAM_CHAT_ID"),
		TelegramHTTPDelay: defaultTelegramDelay,
	}
}

func newApp(cfg Config) (*App, error) {
	logFile, err := os.OpenFile(cfg.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	mlConn, mlClient, err := newMLClient(cfg.MLRPCAddr)
	if err != nil {
		_ = logFile.Close()
		return nil, err
	}

	app := &App{
		cfg:        cfg,
		logFile:    logFile,
		mlConn:     mlConn,
		mlClient:   mlClient,
		httpClient: &http.Client{Timeout: cfg.TelegramHTTPDelay},
	}

	if app.cfg.TelegramBotToken == "" || app.cfg.TelegramChatID == "" {
		log.Printf("telegram alerts disabled: TELEGRAM_BOT_TOKEN and/or TELEGRAM_CHAT_ID not set")
	}

	return app, nil
}
