package permission

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sso/internal/lib/logger/sl"
	"sso/internal/storage"
)

var (
	ErrInvalidInput       = errors.New("input is not found (permission)")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type PermProvider interface {
	GetUserPermissions(ctx context.Context, userID int64) ([]string, error)
	HasUserPermission(ctx context.Context, userID int64, permission string) (bool, error)
}

type Permission struct {
	log          *slog.Logger
	permProvider PermProvider
}

func New(log *slog.Logger, permissionProvider PermProvider) *Permission {
	return &Permission{
		log:          log,
		permProvider: permissionProvider,
	}
}

func (p *Permission) GetUserPermissions(ctx context.Context, userID int64) ([]string, error) {
	const op = "Permission.GetUserPermissions"

	log := p.log.With(
		slog.String("op", op),
		slog.Int64("userID", userID),
	)

	log.Info("retrieving permissions")

	perms, err := p.permProvider.GetUserPermissions(ctx, userID)
	if err != nil {
		log.Error("failed to get user permissions", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return perms, nil
}

func (p *Permission) HasUserPermission(
	ctx context.Context,
	userID int64,
	permission string,
) (bool, error) {
	const op = "Permission.HasUserPermission"

	log := p.log.With(
		slog.String("op", op),
		slog.Int64("userID", userID),
		slog.String("permission", permission),
	)

	log.Info("checking if user has this permission")

	allowed, err := p.permProvider.HasUserPermission(ctx, userID, permission)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			p.log.Warn("user not found", sl.Err(err))

			return false, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		} else if errors.Is(err, storage.ErrPermissionNotFound) {
			p.log.Warn("permission not found", sl.Err(err))

			return false, fmt.Errorf("%s: %w", op, ErrInvalidInput)
		}

		p.log.Error("failed to check whether user has this permission or not", sl.Err(err))

		return false, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("checked the permission for user", slog.Bool("allowed", allowed))

	return allowed, nil
}
