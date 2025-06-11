package permission

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sso/internal/domain/models"
	"sso/internal/lib/logger/sl"
	"sso/internal/services/user"
	"sso/internal/storage"
)

var (
	ErrInvalidInput       = errors.New("input is not found (permission)")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrPermissionDenied   = errors.New("permission denied")
)

type PermissionRepository interface {
	GetUserPermissions(ctx context.Context, userID int64) ([]string, error)
	UserPermissions(ctx context.Context, userID int64) ([]models.Permission, error)
	HasUserPermission(ctx context.Context, userID int64, permission string) (bool, error)
	AddUserPermission(ctx context.Context, userID int64, permissionID int64) error
}

type PermProvider interface {
	GetUserPermissions(ctx context.Context, userID int64) ([]string, error)
	GetUserPermissionsAsModels(ctx context.Context, userID int64) ([]models.Permission, error)
	HasUserPermission(ctx context.Context, userID int64, permission string) (bool, error)
	ValidateUserAccess(ctx context.Context, userID int64, requiredPermissions []string) error
	GrantPermission(ctx context.Context, userID int64, permissionID int64) error
}

type Permission struct {
	log      *slog.Logger
	permRepo PermissionRepository
	userRepo user.UserRepository
	//permProvider PermProvider
}

func New(log *slog.Logger, permissionRepo PermissionRepository, userRepo user.UserRepository) *Permission {
	return &Permission{
		log:      log,
		permRepo: permissionRepo,
		userRepo: userRepo,
	}
}

func (p *Permission) GetUserPermissions(ctx context.Context, userID int64) ([]string, error) {
	const op = "Permission.GetUserPermissions"

	log := p.log.With(
		slog.String("op", op),
		slog.Int64("userID", userID),
	)

	log.Info("retrieving user permissions")

	if err := p.validateUserExists(ctx, userID); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	perms, err := p.permRepo.GetUserPermissions(ctx, userID)
	if err != nil {
		log.Error("failed to get user permissions", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("retrieved user permissions", slog.Int("count", len(perms)))
	return perms, nil
}

func (p *Permission) GetUserPermissionsAsModels(ctx context.Context, userID int64) ([]models.Permission, error) {
	const op = "Permission.GetUserPermissionsAsModels"

	log := p.log.With(
		slog.String("op", op),
		slog.Int64("userID", userID),
	)

	log.Info("retrieving user permissions as models")

	// Валидация пользователя
	if err := p.validateUserExists(ctx, userID); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	perms, err := p.permRepo.UserPermissions(ctx, userID)
	if err != nil {
		log.Error("failed to get user permissions", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("retrieved user permissions as models", slog.Int("count", len(perms)))
	return perms, nil
}

// HasUserPermission checks if user has permission
// returns true if user has permission
// returns false if user does not have permission
// returns error if permission does not exist
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

	if permission == "" {
		return false, fmt.Errorf("%s: %w", op, ErrInvalidInput)
	}

	if err := p.validateUserExists(ctx, userID); err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	allowed, err := p.permRepo.HasUserPermission(ctx, userID, permission)
	if err != nil {
		if errors.Is(err, storage.ErrPermissionNotFound) {
			log.Warn("permission not found", sl.Err(err))
			return false, fmt.Errorf("%s: %w", op, ErrInvalidInput)
		}
		log.Error("failed to check user permission", sl.Err(err))
		return false, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("checked the permission for user", slog.Bool("allowed", allowed))
	return allowed, nil
}

// ValidateUserAccess validates user access to app
// requiredPermissions - list of required permissions
// returns error if user lacks required permissions
// returns nil if user has all required permissions
// returns nil if no required permissions are specified
func (p *Permission) ValidateUserAccess(ctx context.Context, userID int64, requiredPermissions []string) error {
	const op = "Permission.ValidateUserAccess"

	if len(requiredPermissions) == 0 {
		return nil // no need for permissions
	}

	log := p.log.With(
		slog.String("op", op),
		slog.Int64("userID", userID),
		slog.Any("requiredPermissions", requiredPermissions),
	)

	log.Info("validating user access")

	userPerms, err := p.GetUserPermissions(ctx, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// map for quick access
	userPermMap := make(map[string]bool)
	for _, perm := range userPerms {
		userPermMap[perm] = true
	}

	// required permissions check
	missingPermissions := make([]string, 0)
	for _, required := range requiredPermissions {
		if !userPermMap[required] {
			missingPermissions = append(missingPermissions, required)
		}
	}

	if len(missingPermissions) > 0 {
		log.Warn("user lacks required permissions", slog.Any("missing", missingPermissions))
		return fmt.Errorf("%s: missing permissions: %v", op, missingPermissions)
	}

	log.Info("user access validated")
	return nil
}

// GrantPermission grants permission to user
func (p *Permission) GrantPermission(ctx context.Context, userID int64, permissionID int64) error {
	const op = "Permission.GrantPermission"

	log := p.log.With(
		slog.String("op", op),
		slog.Int64("userID", userID),
		slog.Int64("permissionID", permissionID),
	)

	log.Info("granting permission to user")

	if err := p.validateUserExists(ctx, userID); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	err := p.permRepo.AddUserPermission(ctx, userID, permissionID)
	if err != nil {
		log.Error("failed to grant permission", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("permission granted to user")
	return nil
}

func (p *Permission) validateUserExists(ctx context.Context, userID int64) error {
	_, err := p.userRepo.UserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return ErrInvalidCredentials
		}
		return err
	}
	return nil
}
