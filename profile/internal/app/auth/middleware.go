package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Claims struct {
	UserID      int64           `json:"user_id"`
	Email       string          `json:"email"`
	Name        string          `json:"name"`
	AppID       int32           `json:"app_id"`
	Permissions json.RawMessage `json:"permissions"` // Use RawMessage to handle different formats
	jwt.RegisteredClaims
}

type Permission struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type JWTValidator struct {
	secretKey []byte
}

func NewJWTValidator(secretKey string) *JWTValidator {
	return &JWTValidator{
		secretKey: []byte(secretKey),
	}
}

// ParsePermissions парсит permissions из разных форматов
func (c *Claims) ParsePermissions() []Permission {
	if len(c.Permissions) == 0 {
		return []Permission{}
	}

	// Попробуем сначала как массив объектов Permission
	var permissionObjects []Permission
	if err := json.Unmarshal(c.Permissions, &permissionObjects); err == nil {
		return permissionObjects
	}

	// Если не получилось, попробуем как массив строк
	var permissionStrings []string
	if err := json.Unmarshal(c.Permissions, &permissionStrings); err == nil {
		permissions := make([]Permission, len(permissionStrings))
		for i, perm := range permissionStrings {
			permissions[i] = Permission{
				ID:   int64(i + 1), // Присваиваем ID по порядку
				Name: perm,
			}
		}
		return permissions
	}

	// Если и это не сработало, попробуем как одну строку
	var singlePermission string
	if err := json.Unmarshal(c.Permissions, &singlePermission); err == nil {
		return []Permission{{ID: 1, Name: singlePermission}}
	}

	// Если ничего не получилось, возвращаем пустой массив
	return []Permission{}
}

// ValidateToken проверяет JWT токен локально
func (v *JWTValidator) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверяем алгоритм подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Проверяем срок действия
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("token expired")
	}

	return claims, nil
}

// AuthInterceptor - gRPC middleware для проверки JWT
func (v *JWTValidator) AuthInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		// Пропускаем публичные эндпоинты
		if v.isPublicEndpoint(info.FullMethod) {
			return handler(ctx, req)
		}

		// Извлекаем токен
		token, err := v.extractToken(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid token")
		}

		// Валидируем токен локально
		claims, err := v.ValidateToken(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token: "+err.Error())
		}

		// Добавляем claims в контекст
		ctx = v.enrichContext(ctx, claims)

		return handler(ctx, req)
	}
}

// enrichContext добавляет данные пользователя в контекст
func (v *JWTValidator) enrichContext(ctx context.Context, claims *Claims) context.Context {
	ctx = context.WithValue(ctx, "user_id", claims.UserID)
	ctx = context.WithValue(ctx, "user_email", claims.Email)
	ctx = context.WithValue(ctx, "user_name", claims.Name)
	ctx = context.WithValue(ctx, "app_id", claims.AppID)

	// Парсим permissions в правильном формате
	permissions := claims.ParsePermissions()
	ctx = context.WithValue(ctx, "permissions", permissions)

	return ctx
}

// extractToken извлекает JWT из gRPC metadata
func (v *JWTValidator) extractToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("missing metadata")
	}

	authHeaders, ok := md["authorization"]
	if !ok || len(authHeaders) == 0 {
		return "", errors.New("missing authorization header")
	}

	authHeader := authHeaders[0]
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.New("invalid authorization header format")
	}

	return parts[1], nil
}

// isPublicEndpoint проверяет публичные эндпоинты
func (v *JWTValidator) isPublicEndpoint(method string) bool {
	publicEndpoints := []string{
		// public endpoins

		//"/grpc.health.v1.Health/Check",
		//"/metrics",
	}

	for _, endpoint := range publicEndpoints {
		if strings.Contains(method, endpoint) {
			return true
		}
	}
	return false
}
