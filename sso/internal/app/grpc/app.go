package grpcapp

import (
	"context"
	"fmt"
	"time"

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

type AppProvider interface {
	GetAppSecret(ctx context.Context, appID int32) (string, error)
}

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       int
}

var sensitiveMethods = map[string]bool{
	"/sso.Auth/Login":    true,
	"/sso.Auth/Register": true,
}

type MethodPermissions struct {
	Required    []string // mandatory permissions
	OneOf       []string // one of given permissions
	RequireAuth bool     // require authentication without specific permissions
}

var methodPermissions = map[string]MethodPermissions{
	"/sso.Auth/GetUserInfo": {
		RequireAuth: true,
	},
	"/sso.Permission/GetUserPermissions": {
		OneOf: []string{"admin", "staff"},
	},
	"/sso.Auth/Logout": {
		RequireAuth: true,
	},
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

func InterceptorPermission(appProvider AppProvider, pp permission.PermProvider) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// check for sensitive methods
		permConfig, requiresPermission := methodPermissions[info.FullMethod]
		if !requiresPermission {
			return handler(ctx, req)
		}

		token, err := extractTokenFromMetadata(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "missing token")
		}

		claims, err := validateTokenWithDynamicSecret(token, appProvider)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		userID := claims.UserID

		if err := checkPermissions(ctx, userID, permConfig, pp, claims); err != nil {
			return nil, err
		}

		ctx = context.WithValue(ctx, "user_claims", claims)
		ctx = context.WithValue(ctx, "user_id", userID)

		return handler(ctx, req)
	}
}

func extractTokenFromMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	authHeader := md.Get("authorization")
	if len(authHeader) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization header")
	}

	token := strings.TrimPrefix(authHeader[0], "Bearer ")
	if token == authHeader[0] {
		return "", status.Error(codes.Unauthenticated, "invalid authorization header format")
	}

	return token, nil
}

func validateTokenWithDynamicSecret(token string, appProvider AppProvider) (*jwt.TokenClaims, error) {
	claims, err := jwt.DecodeWithoutValidation(token)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	secret, err := appProvider.GetAppSecret(ctx, claims.AppID)
	if err != nil {
		return nil, fmt.Errorf("failed to get app secret: %w", err)
	}

	validClaims, err := jwt.ValidateToken(token, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}

	return validClaims, nil
}

func checkPermissions(
	ctx context.Context,
	userID int64,
	permConfig MethodPermissions,
	pp permission.PermProvider,
	claims *jwt.TokenClaims,
) error {
	// only authentication
	if permConfig.RequireAuth && len(permConfig.Required) == 0 && len(permConfig.OneOf) == 0 {
		return nil
	}

	// check for required permissions
	if len(permConfig.Required) > 0 {
		for _, reqiredPerm := range permConfig.Required {
			if claims.HasPermission(reqiredPerm) {
				continue
			}

			// if not in token, check in DB
			allowed, err := pp.HasUserPermission(ctx, userID, reqiredPerm)
			if err != nil {
				return status.Error(codes.Internal, "failed to check user permissions")
			}
			if !allowed {
				return status.Error(codes.PermissionDenied, fmt.Sprintf("missing required permission: %s", reqiredPerm))
			}
		}
	}

	// check for one of permissions
	if len(permConfig.OneOf) > 0 {
		hasAny := false

		// first in token
		for _, perm := range permConfig.OneOf {
			if claims.HasPermission(perm) {
				hasAny = true
				break
			}
		}

		// DB check
		if !hasAny {
			for _, perm := range permConfig.OneOf {
				allowed, err := pp.HasUserPermission(ctx, userID, perm)
				if err != nil {
					return status.Error(codes.Internal, "failed to check user permissions")
				}
				if allowed {
					hasAny = true
					break
				}
			}
		}
		if !hasAny {
			return status.Error(codes.PermissionDenied, fmt.Sprintf("missing one of required permissions: %s", permConfig.OneOf))
		}
	}

	return nil
}

func New(
	log *slog.Logger,
	authService authgrpc.Auth,
	permissionService authgrpc.PermissionService,
	appProvider AppProvider,
	permProvider permission.PermProvider,
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
		InterceptorLogging(log),
		InterceptorPermission(appProvider, permProvider),
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

func GetUserClaimsFromContext(ctx context.Context) (*jwt.TokenClaims, bool) {
	claims, ok := ctx.Value("user_claims").(*jwt.TokenClaims)
	return claims, ok
}

func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value("user_id").(int64)
	return userID, ok
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
