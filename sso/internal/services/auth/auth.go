package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	_ "sso/internal/config"
	"sso/internal/domain/models"
	"sso/internal/lib/jwt"
	"sso/internal/lib/logger/sl"
	"sso/internal/lib/mailer"
	"sso/internal/services"
	"sso/internal/storage"
	"sso/internal/validator"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type UserSaver interface {
	SaveUserWithPermission(
		ctx context.Context,
		name string,
		phone string,
		address string,
		email string,
		passwordHash []byte,
		permissionID int64,
	) (uid int64, resName string, resEmail string, activated bool, err error)
	//AddUserPermission(ctx context.Context, userID int64, permissionID int64) error
	SaveVerificationToken(ctx context.Context, token string, userID int64, expiresAt time.Time) error
	VerifyEmail(ctx context.Context, token string) (userID int64, err error)
	GetVerificationTokenByUserID(ctx context.Context, userID int64) (string, time.Time, error)
}

type RefreshSaver interface {
	SaveRefresh(ctx context.Context, refresh string, userID int64, appID int32, expiresAt time.Time) error
	DeleteRefresh(ctx context.Context, token string) error
	ExistsRefresh(ctx context.Context, token string) (bool, error)
}

type UserProvider interface {
	UserByID(ctx context.Context, userID int64) (models.User, error)
	UserByEmail(ctx context.Context, email string) (models.User, error)
}

type AppProvider interface {
	App(ctx context.Context, appID int32) (models.App, error)
	GetAppSecret(ctx context.Context, appID int32) (string, error)
}

type PermProvider interface {
	GetUserPermissionsAsModels(ctx context.Context, userID int64) ([]models.Permission, error)
}

type Auth struct {
	log          *slog.Logger
	usrSaver     UserSaver
	refreshSaver RefreshSaver
	usrProvider  UserProvider
	appProvider  AppProvider
	permProvider PermProvider
	emailClient  *mailer.MailtrapClient
	baseURL      string
	tokenTTL     time.Duration
	refreshTTL   time.Duration
}

func New(
	log *slog.Logger,
	userSaver UserSaver,
	refreshSaver RefreshSaver,
	userProvider UserProvider,
	appProvider AppProvider,
	permProvider PermProvider,
	emailClient *mailer.MailtrapClient,
	baseURL string,
	tokenTTL time.Duration,
	refreshTTL time.Duration,
) *Auth {
	return &Auth{
		log:          log,
		usrSaver:     userSaver,
		refreshSaver: refreshSaver,
		usrProvider:  userProvider,
		appProvider:  appProvider,
		permProvider: permProvider,
		emailClient:  emailClient,
		baseURL:      baseURL,
		tokenTTL:     tokenTTL,
		refreshTTL:   refreshTTL,
	}
}

// RegisterNewUser registers a new user and returns ID, returns error if email already exists
func (a *Auth) RegisterNewUser(ctx context.Context,
	name string,
	phone string,
	address string,
	email string,
	password string,
) (int64, string, string, bool, error) {
	// op - name of the current package and function; convenient to put in logs and errors to find problems faster
	const op = "Auth.RegisterNewUser"
	
	log := a.log.With(
		slog.String("op", op),
		slog.String("name", name),
		slog.String("phone", phone),
		slog.String("address", address),
		slog.String("email", email),
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
	id, name, email, activated, err := a.usrSaver.SaveUserWithPermission(ctx, name, phone, address, email, passwordHash, 1)
	if err != nil {
		log.Error("failed to save user with permission", sl.Err(err))
		
		return 0, "", "", false, fmt.Errorf("%s: %w", op, err)
	}
	
	//if err := a.usrSaver.AddUserPermission(ctx, id, 1); err != nil {
	//	log.Error("failed to assign user permission", sl.Err(err))
	//
	//	return 0, "", "", false, fmt.Errorf("%s: %w", op, err)
	//}
	
	log.Info("user registered successfully", slog.Int64("userID", id))
	
	return id, name, email, activated, nil
}

func (a *Auth) generateVerificationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (a *Auth) SendVerificationEmail(ctx context.Context, userId int64, appID int32) (bool, string, int64, error) {
	const op = "Auth.SendVerificationEmail"
	
	log := a.log.With(
		slog.String("op", op),
		slog.Int64("userID", userId),
		slog.Int("appID", int(appID)),
	)
	
	log.Info("sending verification email")
	
	user, err := a.usrProvider.UserByID(ctx, userId)
	if err != nil {
		log.Error("failed to get user", sl.Err(err))
		return false, "User not found", 0, fmt.Errorf("%s: %w", op, err)
	}
	
	if user.Activated {
		log.Info("user already activated")
		return false, "User already activated", 0, nil
	}
	
	token, err := a.generateVerificationToken()
	if err != nil {
		log.Error("failed to generate verification token", sl.Err(err))
		return false, "Failed to generate verification token", 0, fmt.Errorf("%s: %w", op, err)
	}
	
	expiresAt := time.Now().Add(24 * time.Hour)
	if err := a.usrSaver.SaveVerificationToken(ctx, token, userId, expiresAt); err != nil {
		log.Error("failed to save verification token", sl.Err(err))
		return false, "Failed to save verification token", 0, fmt.Errorf("%s: %w", op, err)
	}
	
	if err := a.emailClient.SendVerificationEmail(ctx, user.Email, user.Name, token, a.baseURL); err != nil {
		log.Error("failed to send verification email", sl.Err(err))
		return false, "Failed to send verification email", 0, fmt.Errorf("%s: %w", op, err)
	}
	
	log.Info("verification email sent successfully")
	
	return true, "Verification email sent successfully", expiresAt.Unix(), nil
}

func (a *Auth) EmailVerify(ctx context.Context, token string) (bool, string, bool, error) {
	const op = "Auth.EmailVerify"
	
	log := a.log.With(
		slog.String("op", op),
	)
	
	log.Info("processing email verification")
	
	if token == "" {
		return false, "Verification token is required", false, fmt.Errorf("%s: empty token", op)
	}
	
	userID, err := a.usrSaver.VerifyEmail(ctx, token)
	if err != nil {
		if errors.Is(err, storage.ErrTokenNotFound) {
			log.Warn("invalid or expired verification token")
			return false, "Invalid or expired verification token", false, fmt.Errorf("%s: %w", op, err)
		}
		log.Error("failed to verify email", sl.Err(err))
		return false, "Failed to verify email", false, fmt.Errorf("%s: %w", op, err)
	}
	
	log.Info("email verified successfully", slog.Int64("userID", userID))
	
	return true, "Email verified successfully", true, nil
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
	
	permissions, err := a.permProvider.GetUserPermissionsAsModels(ctx, user.ID)
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
	
	user, err := a.usrProvider.UserByID(ctx, userID)
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
	
	permissions, err := a.permProvider.GetUserPermissionsAsModels(ctx, user.ID)
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

func (a *Auth) ForgotPassword(ctx context.Context, email string, appID int32) (bool, string, int64, error) {
	const op = "Auth.ForgotPassword"
	
	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email),
		slog.Int("appID", int(appID)),
	)
	
	log.Info("processing forgot password request")
	
	// TODO: Implement forgot password logic
	// For now, return not implemented
	log.Warn("forgot password not implemented yet")
	
	return false, "Forgot password functionality is not implemented yet", 0, fmt.Errorf("%s: not implemented", op)
}

func (a *Auth) ResetPassword(ctx context.Context, token string, password string) (bool, string, error) {
	const op = "Auth.ResetPassword"
	
	log := a.log.With(
		slog.String("op", op),
	)
	
	log.Info("processing reset password request")
	
	// TODO: Implement reset password logic
	// For now, return not implemented
	log.Warn("reset password not implemented yet")
	
	return false, "Reset password functionality is not implemented yet", fmt.Errorf("%s: not implemented", op)
}
