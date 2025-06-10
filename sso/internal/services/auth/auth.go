package auth

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	_ "sso/internal/config"
	"sso/internal/domain/models"
	"sso/internal/lib/jwt"
	"sso/internal/lib/logger/sl"
	"sso/internal/services"
	"sso/internal/storage"
	"sso/internal/validator"
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
		name string,
		phone string,
		address string,
	) (uid int64, resName string, resEmail string, activated bool, err error)
	AddUserPermission(ctx context.Context, userID int64, permissionID int64) error
}

type RefreshSaver interface {
	SaveRefresh(ctx context.Context, refresh string, userID int64, appID int32, expiresAt time.Time) error
	DeleteRefresh(ctx context.Context, token string) error
	ExistsRefresh(ctx context.Context, token string) (bool, error)
}

type UserProvider interface {
	User(ctx context.Context,
		email string,
		name string,
		phone string,
		address string,
		activated bool,
	) (models.User, error)
	UserByID(ctx context.Context, userID int64) (models.User, error)
	UserByEmail(ctx context.Context, email string) (models.User, error)
}

type AppProvider interface {
	App(ctx context.Context, appID int32) (models.App, error)
	GetAppSecret(ctx context.Context, appID int32) (string, error)
}

type PermissionProvider interface {
	UserPermissions(ctx context.Context, userID int64) ([]models.Permission, error)
}

type Auth struct {
	log          *slog.Logger
	usrSaver     UserSaver
	refreshSaver RefreshSaver
	usrProvider  UserProvider
	appProvider  AppProvider
	permProvider PermissionProvider
	tokenTTL     time.Duration
	refreshTTL   time.Duration
}

func New(
	log *slog.Logger,
	userSaver UserSaver,
	refreshSaver RefreshSaver,
	userProvider UserProvider,
	appProvider AppProvider,
	permissionProvider PermissionProvider,
	tokenTTL time.Duration,
	refreshTTL time.Duration,
) *Auth {
	return &Auth{
		log:          log,
		usrSaver:     userSaver,
		refreshSaver: refreshSaver,
		usrProvider:  userProvider,
		appProvider:  appProvider,
		permProvider: permissionProvider,
		tokenTTL:     tokenTTL,
		refreshTTL:   refreshTTL,
	}
}

// Registers a new user and returns ID, returns error if email already exists
func (a *Auth) RegisterNewUser(ctx context.Context,
	email string,
	password string,
	name string,
	phone string,
	address string,
) (int64, string, string, bool, error) {
	// op - name of the current package and function; convenient to put in logs and errors to find problems faster
	const op = "Auth.RegisterNewUser"

	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email),
		slog.String("name", name),
		slog.String("phone", phone),
		slog.String("address", address),
	)

	log.Info("registering user")

	v := validator.New()
	validator.ValidateUserEmail(v, email)
	if !v.Valid() {
		log.Error("invalid user email", slog.Any("errors", v.Errors))
		return 0, "", "", false, fmt.Errorf("%s: validation error: %v", op, services.ErrInvalidEmail)
	}
	validator.ValidateUserPassword(v, password)
	if !v.Valid() {
		log.Error("invalid user password", slog.Any("errors", v.Errors))
		return 0, "", "", false, fmt.Errorf("%s: validation error: %v", op, services.ErrInvalidPassword)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))

		return 0, "", "", false, fmt.Errorf("%s: %w", op, err)
	}

	// saving user in DB
	id, name, email, activated, err := a.usrSaver.SaveUser(ctx, email, passwordHash, name, phone, address)
	if err != nil {
		log.Error("failed to save user", sl.Err(err))

		return 0, "", "", false, fmt.Errorf("%s: %w", op, err)
	}

	if err := a.usrSaver.AddUserPermission(ctx, id, 1); err != nil {
		log.Error("failed to assign user permission", sl.Err(err))

		return 0, "", "", false, fmt.Errorf("%s: %w", op, err)
	}

	return id, name, email, activated, nil
}

func (a *Auth) Login(
	ctx context.Context,
	email string,
	password string,
	appID int32,
) (string, string, int64, error) {
	const op = "Auth.Login"

	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email),
		slog.Int("appID", int(appID)),
	)

	log.Info("attempting to login user")

	user, err := a.usrProvider.UserByEmail(ctx, email)
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

	permissions, err := a.permProvider.UserPermissions(ctx, user.ID)
	if err != nil {
		a.log.Error("failed to get user permissions", sl.Err(err))
		return "", "", 0, fmt.Errorf("%s: %w", op, err)
	}

	token, err := jwt.NewToken(user, app, permissions, a.tokenTTL)
	if err != nil {
		a.log.Error("failed to generate access token", sl.Err(err))

		return "", "", 0, fmt.Errorf("%s: %w", op, err)
	}

	refresh, err := jwt.NewRefreshToken(user, app, a.refreshTTL)
	if err != nil {
		a.log.Error("failed to generate refresh token", sl.Err(err))

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

	log.Info("user logged in successfully")

	return token, refresh, expiresAt, nil
}

func (a *Auth) Logout(ctx context.Context, refresh string) (bool, error) {
	const op = "Auth.Logout"
	err := a.refreshSaver.DeleteRefresh(ctx, refresh)
	if err != nil {
		a.log.Error("failed to delete refresh token", slog.String("op", op), sl.Err(err))
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return true, nil
}

func (a *Auth) GetUserInfo(ctx context.Context, token string) (int64, string, string, string, string, bool, error) {
	const op = "Auth.GetUserInfo"

	claims, err := jwt.DecodeWithoutValidation(token)
	if err != nil {
		a.log.Error("failed to decode token", slog.String("op", op), sl.Err(err))
		return 0, "", "", "", "", false, fmt.Errorf("%s: %w", op, err)
	}

	appID := claims.AppID

	app, err := a.appProvider.App(ctx, appID)
	if err != nil {
		a.log.Error("failed to get app", slog.String("op", op), sl.Err(err))
		return 0, "", "", "", "", false, fmt.Errorf("%s: %w", op, err)
	}

	validClaims, err := jwt.ValidateToken(token, app.Secret)
	if err != nil {
		a.log.Error("invalid token", slog.String("op", op), sl.Err(err))
		return 0, "", "", "", "", false, fmt.Errorf("%s: %w", op, err)
	}

	userID := validClaims.UserID

	user, err := a.usrProvider.UserByID(ctx, int64(userID))
	if err != nil {
		a.log.Error("failed to get user", slog.String("op", op), sl.Err(err))
		return 0, "", "", "", "", false, fmt.Errorf("%s: %w", op, err)
	}

	return user.ID, user.Email, user.Name, user.Phone, user.Address, user.Activated, nil
}

func (a *Auth) RefreshToken(ctx context.Context, refresh string) (string, string, int64, error) {
	const op = "Auth.RefreshTokens"

	claims, err := jwt.DecodeWithoutValidation(refresh)
	if err != nil {
		a.log.Error("failed to decode token", slog.String("op", op), sl.Err(err))
		return "", "", 0, fmt.Errorf("%s: %w", op, err)
	}

	appID := claims.AppID

	app, err := a.appProvider.App(ctx, appID)
	if err != nil {
		a.log.Error("failed to get app", slog.String("op", op), sl.Err(err))
		return "", "", 0, fmt.Errorf("%s: %w", op, err)
	}

	validClaims, err := jwt.ValidateRefreshToken(refresh, app.Secret)
	if err != nil {
		a.log.Error("invalid refresh token", slog.String("op", op), sl.Err(err))
	}

	userID := validClaims.UserID

	user, err := a.usrProvider.UserByID(ctx, userID)
	if err != nil {
		return "", "", 0, fmt.Errorf("%s: %w", op, err)
	}

	permissions, err := a.permProvider.UserPermissions(ctx, user.ID)
	if err != nil {
		a.log.Error("failed to get user permissions", sl.Err(err))
		return "", "", 0, fmt.Errorf("%s: %w", op, err)
	}

	newToken, err := jwt.NewToken(user, app, permissions, a.tokenTTL)
	if err != nil {
		return "", "", 0, fmt.Errorf("%s: %w", op, err)
	}

	newRefresh, err := jwt.NewRefreshToken(user, app, a.refreshTTL)
	if err != nil {
		return "", "", 0, fmt.Errorf("%s: %w", op, err)
	}

	expiresAt := time.Now().Add(a.tokenTTL).Unix()

	err = a.refreshSaver.DeleteRefresh(ctx, refresh)
	if err != nil {
		a.log.Warn("failed to delete refresh token", slog.String("op", op), sl.Err(err))
	}

	err = a.refreshSaver.SaveRefresh(ctx, newRefresh, user.ID, app.ID, time.Now().Add(a.refreshTTL))
	if err != nil {
		return "", "", 0, fmt.Errorf("%s: %w", op, err)
	}

	return newToken, newRefresh, expiresAt, nil
}

func (a *Auth) GetAppSecret(ctx context.Context, appID int32) (string, error) {
	const op = "Auth.GetAppSecret"
	app, err := a.appProvider.App(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	return app.Secret, nil
}
