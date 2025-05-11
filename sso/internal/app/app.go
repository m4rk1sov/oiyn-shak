package app

import (
	"log/slog"
	grpcapp "sso/internal/app/grpc"
	"sso/internal/services/auth"
	"sso/internal/services/permission"
	"sso/internal/storage/postgres"
	"time"
)

type App struct {
	GRPCServer *grpcapp.App
	Storage    *postgres.Storage
}

func New(
	log *slog.Logger,
	grpcPort int,
	dsn string,
	tokenTTL time.Duration,
	refreshTTL time.Duration,
) *App {
	storage, err := postgres.New(dsn)
	if err != nil {
		panic(err)
	}

	authService := auth.New(log, storage, storage, storage, tokenTTL, refreshTTL)
	permissionService := permission.New(log, storage)

	grpcApp := grpcapp.New(log, authService, permissionService, grpcPort)

	return &App{
		GRPCServer: grpcApp,
	}
}

func (a *App) Stop() {
	if a.GRPCServer != nil {
		a.GRPCServer.Stop()
	}
	if a.Storage != nil {
		a.Storage.Close()
	}
}
