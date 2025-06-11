package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sso/internal/domain/models"
	"sso/internal/lib/logger/sl"
	"sso/internal/storage"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type UserRepository interface {
	UserByID(ctx context.Context, userID int64) (models.User, error)
	UserByEmail(ctx context.Context, email string) (models.User, error)
	SaveUser(ctx context.Context, name, phone, address, email string, passwordHash []byte) (int64, string, string, bool, error)
}

type UserProvider interface {
	GetUserByID(ctx context.Context, userID int64) (models.User, error)
	GetUserByEmail(ctx context.Context, email string) (models.User, error)
	ValidateUserCredentials(ctx context.Context, email, password string) (models.User, error)
}

type User struct {
	log      *slog.Logger
	userRepo UserRepository
}

func New(log *slog.Logger, userRepo UserRepository) *User {
	return &User{
		log:      log,
		userRepo: userRepo,
	}
}

func (u *User) GetUserByID(ctx context.Context, userID int64) (models.User, error) {
	const op = "User.GetUserByID"

	log := u.log.With(
		slog.String("op", op),
		slog.Int64("userID", userID),
	)

	log.Info("getting user by ID")

	user, err := u.userRepo.UserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Warn("user not found")
			return models.User{}, fmt.Errorf("%s: %w", op, ErrUserNotFound)
		}
		log.Error("failed to get user", sl.Err(err))
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user retrieved successfully")
	return user, nil
}

func (u *User) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	const op = "User.GetUserByEmail"

	log := u.log.With(
		slog.String("op", op),
		slog.String("email", email),
	)

	log.Info("getting user by email")

	user, err := u.userRepo.UserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Warn("user not found")
			return models.User{}, fmt.Errorf("%s: %w", op, ErrUserNotFound)
		}
		log.Error("failed to get user", sl.Err(err))
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user retrieved successfully")
	return user, nil
}
