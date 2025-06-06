package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	grpcapp "sso/internal/app/grpc"
	"sso/internal/lib/logger/sl"
	"sso/internal/services/auth"
	"sso/internal/services/permission"
	"sso/internal/storage/postgres"
	"syscall"
	"time"

	httpserver "sso/internal/http"
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
	//swaggerSpec []byte,
	dsn string,
	tokenTTL time.Duration,
	refreshTTL time.Duration,
) *App {
	storage, err := postgres.New(dsn)
	if err != nil {
		panic(err)
	}

	authService := auth.New(log, storage, storage, storage, storage, tokenTTL, refreshTTL)
	permissionService := permission.New(log, storage)

	grpcApp := grpcapp.New(log, authService, permissionService, grpcPort)
	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)
	httpServer := httpserver.NewServer(grpcAddr, httpPort)

	return &App{
		GRPCServer: grpcApp,
		HTTPServer: httpServer,
	}
}

func (a *App) Stop() {
	const op = "app.Stop"

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
		a.Storage.Close()
		a.log.Info("closed database connection")
	}
}
