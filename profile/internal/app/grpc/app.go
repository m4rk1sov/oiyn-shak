package grpcapp

import (
	"fmt"
	"profile/internal/grpc/profile"
	
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
	"net"
	profileService "profile/internal/services/profile"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       int
}

func New(
	log *slog.Logger,
	profileService *profileService.Service,
	port int,
) *App {
	// gRPCServer and connect interceptors
	recoveryOpts := []recovery.Option{
		recovery.WithRecoveryHandler(func(p interface{}) (err error) {
			log.Error("Recovered from panic", slog.Any("panic", p))
			return status.Errorf(codes.Internal, "Internal error")
		}),
	}
	
	gRPCServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		recovery.UnaryServerInterceptor(recoveryOpts...),
	))
	
	// register the service
	profile.Register(gRPCServer, profileService, log)
	
	// return App object with necessary fields
	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		port:       port,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Run() error {
	const op = "grpcApp.Run"
	
	log := a.log.With(
		slog.String("op", op),
		slog.Int("port", a.port),
	)
	
	// Listener for TCP connections
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	
	log.Info("grpc server started", slog.String("addr", l.Addr().String()))
	
	if err := a.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	
	return nil
}

func (a *App) Stop() {
	const op = "grpcApp.Stop"
	
	a.log.With(slog.String("op", op)).
		Info("stopping grpc server", slog.Int("port", a.port))
	
	a.gRPCServer.GracefulStop()
}
