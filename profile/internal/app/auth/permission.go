package auth

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HasPermission проверяет наличие разрешения у пользователя
func HasPermission(ctx context.Context, requiredPermission string) bool {
	permissions, ok := ctx.Value("permissions").([]Permission)
	if !ok {
		// Логируем для отладки
		slog.Debug("permissions not found in context or wrong type",
			slog.String("required_permission", requiredPermission))
		return false
	}

	slog.Debug("checking permission",
		slog.String("required", requiredPermission),
		slog.Int("available_count", len(permissions)))

	for _, perm := range permissions {
		slog.Debug("checking against permission",
			slog.String("permission_name", perm.Name),
			slog.Int64("permission_id", perm.ID))

		if perm.Name == requiredPermission {
			slog.Debug("permission granted", slog.String("permission", requiredPermission))
			return true
		}
	}

	slog.Debug("permission denied",
		slog.String("required", requiredPermission),
		slog.Int("checked_permissions", len(permissions)))
	return false
}

// RequirePermission - middleware для проверки конкретного разрешения
func RequirePermission(permission string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		if !HasPermission(ctx, permission) {
			return nil, status.Error(codes.PermissionDenied,
				fmt.Sprintf("insufficient permissions: %s required", permission))
		}

		return handler(ctx, req)
	}
}

// RequireOwnership проверяет, что пользователь обращается к своему профилю
func RequireOwnership() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		userID, ok := ctx.Value("user_id").(int64)
		if !ok {
			return nil, status.Error(codes.Internal, "user context missing")
		}

		// Извлекаем запрашиваемый userID из request (если есть)
		if userRequest, ok := req.(interface{ GetUserId() int64 }); ok {
			requestedUserID := userRequest.GetUserId()
			if userID != requestedUserID {
				return nil, status.Error(codes.PermissionDenied,
					"access denied: can only access own data")
			}
		}

		return handler(ctx, req)
	}
}

// GetUserFromContext - helper для получения пользователя из контекста
func GetUserFromContext(ctx context.Context) (UserContext, error) {
	userID, ok := ctx.Value("user_id").(int64)
	if !ok {
		return UserContext{}, fmt.Errorf("user_id not found in context")
	}

	email, _ := ctx.Value("email").(string)
	name, _ := ctx.Value("name").(string)
	appID, _ := ctx.Value("app_id").(int32)
	permissions, _ := ctx.Value("permissions").([]Permission)

	return UserContext{
		UserID:      userID,
		Email:       email,
		Name:        name,
		AppID:       appID,
		Permissions: permissions,
	}, nil
}

func DebugUserContext(ctx context.Context) {
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		slog.Error("failed to get user context", slog.String("error", err.Error()))
		return
	}

	slog.Info("user context debug",
		slog.Int64("user_id", userCtx.UserID),
		slog.String("email", userCtx.Email),
		slog.Int("permissions_count", len(userCtx.Permissions)))

	for i, perm := range userCtx.Permissions {
		slog.Info("permission",
			slog.Int("index", i),
			slog.Int64("id", perm.ID),
			slog.String("name", perm.Name))
	}
}

type UserContext struct {
	UserID      int64
	Email       string
	Name        string
	AppID       int32
	Permissions []Permission
}
