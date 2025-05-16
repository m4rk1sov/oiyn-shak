package auth

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"sso/internal/domain/models"
	"sso/internal/lib/jwt"
	"sso/internal/lib/logger/sl"
	"sso/internal/storage"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type UserSaver interface {
	SaveUser(
		ctx context.Context,
		email string,
		passwordHash []byte,
	) (uid int64, err error)
	AddUserPermission(ctx context.Context, userID int64, permissionID int64) error
}

type RefreshSaver interface {
	SaveRefresh(ctx context.Context, refresh string, userID int64, appID int, expiresAt time.Time) error
	DeleteRefresh(ctx context.Context, token string) error
	ExistsRefresh(ctx context.Context, token string) (bool, error)
}

type UserProvider interface {
	User(ctx context.Context, email string) (models.User, error)
}

type AppProvider interface {
	App(ctx context.Context, appID int) (models.App, error)
}

type Auth struct {
	log          *slog.Logger
	usrSaver     UserSaver
	refreshSaver RefreshSaver
	usrProvider  UserProvider
	appProvider  AppProvider
	tokenTTL     time.Duration
	refreshTTL   time.Duration
}

func New(
	log *slog.Logger,
	userSaver UserSaver,
	refreshSaver RefreshSaver,
	userProvider UserProvider,
	appProvider AppProvider,
	tokenTTL time.Duration,
	refreshTTL time.Duration,
) *Auth {
	return &Auth{
		log:          log,
		usrSaver:     userSaver,
		refreshSaver: refreshSaver,
		usrProvider:  userProvider,
		appProvider:  appProvider,
		tokenTTL:     tokenTTL,
		refreshTTL:   refreshTTL,
	}
}

// Registers a new user and returns ID, returns error if email already exists
func (a *Auth) RegisterNewUser(ctx context.Context, email string, password string) (int64, error) {
	// op - name of current package and function. Convenient to put in logs and errors, to find problems faster
	const op = "Auth.RegisterNewUser"

	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email),
	)

	log.Info("registering user")

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	// saving user in DB
	id, err := a.usrSaver.SaveUser(ctx, email, passwordHash)
	if err != nil {
		log.Error("failed to save user", sl.Err(err))

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	if err := a.usrSaver.AddUserPermission(ctx, id, 1); err != nil {
		log.Error("failed to assign user permission", sl.Err(err))

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (a *Auth) Login(
	ctx context.Context,
	email string,
	password string,
	appID int,
) (string, string, int64, error) {
	const op = "Auth.Login"

	log := a.log.With(
		slog.String("op", op),
		slog.String("username", email),
	)

	log.Info("attempting to login user")

	user, err := a.usrProvider.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			a.log.Warn("user not found", sl.Err(err))

			return "", "", 0, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}

		a.log.Error("failed to get user", sl.Err(err))

		return "", "", 0, fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
		a.log.Info("invalid credentials", sl.Err(err))

		return "", "", 0, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	// info about app
	app, err := a.appProvider.App(ctx, appID)
	if err != nil {
		return "", "", 0, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user logged in successfully")

	token, err := jwt.NewToken(user, app, a.tokenTTL)
	refresh, err := jwt.NewRefreshToken(user, app, a.refreshTTL)
	if err != nil {
		a.log.Error("failed to generate token", sl.Err(err))

		return "", "", 0, fmt.Errorf("%s: %w", op, err)
	}
	expiresAt := time.Now().Add(a.tokenTTL).Unix()

	return token, refresh, expiresAt, nil
}

func (a *Auth) Logout(ctx context.Context, token string) (bool, error) {
	const op = "Auth.Logout"

	err := a.refreshSaver.DeleteRefresh(ctx, token)
	if err != nil {
		a.log.Error("failed to delete refresh token", slog.String("op", op), sl.Err(err))
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return true, nil
}

func (a *Auth) GetUserInfo(ctx context.Context, token string) (int64, string, error) {
	const op = "Auth.GetUserInfo"

	claims, err := jwt.ValidateToken(token, "")
	if err != nil {
		a.log.Error("invalid token", slog.String("op", op), sl.Err(err))
		return 0, "", fmt.Errorf("%s: %w", op, err)
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, "", fmt.Errorf("%s: user_id not found in token", op)
	}

	email, ok := claims["email"].(string)
	if !ok {
		return 0, "", fmt.Errorf("%s: email not found in token", op)
	}

	return int64(userID), email, nil
}

func (a *Auth) RefreshToken(ctx context.Context, refresh string) (string, string, int64, error) {
	const op = "Auth.RefreshTokens"

	claims, err := jwt.ValidateRefreshToken(refresh, "")
	if err != nil {
		a.log.Error("invalid refresh token", slog.String("op", op), sl.Err(err))
	}

	_, ok := claims["user_id"].(float64)
	if !ok {
		return "", "", 0, fmt.Errorf("%s: missing user_id", op)
	}

	appID, ok := claims["app_id"].(float64)
	if !ok {
		return "", "", 0, fmt.Errorf("%s: missing app_id", op)
	}

	user, err := a.usrProvider.User(ctx, claims["email"].(string))
	if err != nil {
		return "", "", 0, fmt.Errorf("%s: %w", op, err)
	}

	app, err := a.appProvider.App(ctx, int(appID))
	if err != nil {
		return "", "", 0, fmt.Errorf("%s: %w", op, err)
	}

	newToken, err := jwt.NewToken(user, app, a.tokenTTL)
	if err != nil {
		return "", "", 0, fmt.Errorf("%s: %w", op, err)
	}

	newRefresh, err := jwt.NewRefreshToken(user, app, a.refreshTTL)
	if err != nil {
		return "", "", 0, fmt.Errorf("%s: %w", op, err)
	}

	expiresAt := time.Now().Add(a.tokenTTL).Unix()

	err = a.refreshSaver.SaveRefresh(ctx, refresh, user.ID, app.ID, time.Now().Add(a.refreshTTL))
	if err != nil {
		return "", "", 0, fmt.Errorf("%s: %w", op, err)
	}

	exists, err := a.refreshSaver.ExistsRefresh(ctx, refresh)
	if err != nil || !exists {
		return "", "", 0, fmt.Errorf("%s: invalid refresh token", op)
	}

	return newToken, newRefresh, expiresAt, nil
}
