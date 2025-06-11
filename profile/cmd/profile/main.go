package main

import (
	"context"
	"errors"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"os"
	"os/signal"
	"profile/internal/app"
	"profile/internal/config"
	"profile/internal/lib/logger/sl"
	"syscall"
	"time"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		slog.Warn("Error loading .env file")
	}
	cfg := config.MustLoad()

	// Initialize logger
	log := setupLogger(cfg.Env)

	// Initialize app
	application := app.New(log, cfg.GRPC.Port, cfg.HTTPServer.Port, cfg.DSN, cfg.JWT.TokenTTL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	grp, grpCtx := errgroup.WithContext(ctx)

	// Launch gRPC
	grp.Go(func() error {
		log.Info("starting gRPC server", slog.Int("port", cfg.GRPC.Port))

		// start a gRPC server in a separate goroutine
		errChan := make(chan error, 1)
		go func() {
			errChan <- application.GRPCServer.Run()
		}()

		// wait for context cancellation or server error
		select {
		case <-grpCtx.Done():
			application.GRPCServer.Stop()
			return grpCtx.Err()
		case err := <-errChan:
			return err
		}
	})

	// Launch HTTP
	grp.Go(func() error {
		log.Info("starting HTTP server", slog.Int("port", cfg.HTTPServer.Port))

		// start an HTTP server
		if err := application.HTTPServer.Start(); err != nil {
			return err
		}

		// wait for context cancellation or server error
		<-grpCtx.Done()

		// graceful shutdown of the HTTP server
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		return application.HTTPServer.Stop(shutdownCtx)
	})

	// Graceful shutdown
	// channel for signal info
	stop := make(chan os.Signal, 1)
	// Listening to the signals
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	// waiting the data from a channel (pkill -2 or SIGTERM)
	go func() {
		<-stop
		log.Info("received signal to stop")
		cancel() // context cancellation in goroutines
	}()

	// wait for all goroutines to finish
	if err := grp.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("server exited with error", sl.Err(err))
	}

	// initiate a graceful shutdown
	err = application.CloseStorage()
	if err != nil {
		log.Error("failed to close storage", sl.Err(err))
	}
	log.Info("Gracefully stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
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
