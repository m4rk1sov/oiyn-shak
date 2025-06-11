package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	grpcapp "profile/internal/app/grpc"
	"profile/internal/lib/logger/sl"
	"profile/internal/services/profile"
	"profile/internal/storage/postgres"
	"syscall"
	"time"

	httpserver "profile/internal/http"
)

type App struct {
	GRPCServer *grpcapp.App
	Storage    *postgres.Storage
	HTTPServer *httpserver.Server
	log        *slog.Logger
}

func New(
	log *slog.Logger,
	grpcPort int,
	httpPort int,
	dsn string,
	tokenTTL time.Duration,
) *App {
	storage, err := postgres.New(dsn, log)
	if err != nil {
		panic(err)
	}

	profileService := profile.New(log, storage)

	grpcApp := grpcapp.New(log, profileService, grpcPort)

	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)
	httpServer := httpserver.NewServer(grpcAddr, httpPort, log)

	return &App{
		GRPCServer: grpcApp,
		HTTPServer: httpServer,
		Storage:    storage,
		log:        log,
	}
}

func (a *App) CloseStorage() error {
	if a.Storage != nil {
		err := a.Storage.Close()
		if err != nil {
			return err
		}
		a.log.Info("closed database connection")
	}
	return nil
}

func (a *App) Stop() {
	const op = "app.Stop"

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	if a.HTTPServer != nil {
		if err := a.HTTPServer.Stop(ctx); err != nil {
			a.log.Error("failed to stop HTTP server", op, sl.Err(err))
		}
	}

	if a.GRPCServer != nil {
		a.GRPCServer.Stop()
		a.log.Info("stopped gRPC server")
	}
	if a.Storage != nil {
		err := a.Storage.Close()
		if err != nil {
			a.log.Error("failed to close storage", sl.Err(err))
		}
		a.log.Info("closed database connection")
	}
}
