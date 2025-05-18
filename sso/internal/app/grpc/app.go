package grpcapp

import (
	"context"
	"fmt"
	//"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log/slog"
	"net"
	authgrpc "sso/internal/grpc/auth"
	"sso/internal/lib/jwt"
	"sso/internal/services/permission"
	"strings"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       int
}

var sensitiveMethods = map[string]bool{
	"/sso.Auth/Login":    true,
	"/sso.Auth/Register": true,
}

func InterceptorLogging(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		// Skip logging for sensitive methods
		if sensitiveMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		log.Info("gRPC request", slog.String("method", info.FullMethod))
		resp, err = handler(ctx, req)
		if err != nil {
			log.Error("gRPC error", slog.String("method", info.FullMethod), slog.String("error", err.Error()))
		} else {
			log.Info("gRPC response", slog.String("method", info.FullMethod))
		}
		return resp, err
	}
}

func InterceptorPermission(secret string, pp permission.PermProvider, permission string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// extracting the metadata from token (HTTP-like)
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		// Getting the token from header
		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing token")
		}
		tokenString := strings.TrimPrefix(authHeader[0], "Bearer ")

		claims, err := jwt.ValidateToken(tokenString, secret)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		userID, ok := claims["user_id"].(float64)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "invalid user id")
		}

		allowed, err := pp.HasUserPermission(ctx, int64(userID), permission)
		if err != nil {
			return nil, status.Error(codes.Internal, "permission lookup failed")
		}

		if allowed {
			return handler(ctx, req)
		}
		return nil, status.Error(codes.PermissionDenied, "missing required permission")
	}
}

func New(
	log *slog.Logger,
	authService authgrpc.Auth,
	permissionService authgrpc.Permission,
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
		//logging.UnaryServerInterceptor(InterceptorLogger(log), loggingOpts...),
		InterceptorLogging(log),
	))

	// register the service Auth
	authgrpc.Register(gRPCServer, authService, permissionService)

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

	// Listener for TCP connections
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	a.log.Info("grpc server started", slog.String("addr", l.Addr().String()))

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
