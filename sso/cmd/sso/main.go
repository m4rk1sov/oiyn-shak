package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log/slog"
	"os"
	"os/signal"
	"sso/internal/app"
	"sso/internal/config"
	"syscall"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
	}
	cfg := config.MustLoad()

	// Initialize logger
	log := setupLogger(cfg.Env)

	// Initialize app
	application := app.New(log, cfg.GRPC.Port, cfg.DSN, cfg.TokenTTL)

	// Launch gRPC
	go func() {
		application.GRPCServer.MustRun()
	}()

	// Graceful shutdown
	// channel for signal info
	stop := make(chan os.Signal, 1)
	// Listening the signals
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	// waiting the data from channel (pkill -2 or SIGTERM)
	<-stop

	// initiate graceful shutdown
	application.Stop()
	log.Info("Gracefully stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}
