package auth

import (
	"context"
	"errors"
	"sso/internal/services"
	"sso/internal/services/auth"
	"sso/internal/storage"
	// codes for grpc clients to understand
	"google.golang.org/grpc/codes"
	// status of errors for understanding of grpc clients
	"google.golang.org/grpc/status"

	ssov1 "github.com/m4rk1sov/protos/gen/go/sso"
)

type authServer struct {
	// to resolve compatibility issues, when adding new proto-generated files
	// will return a not implemented error
	ssov1.UnimplementedAuthServer
	auth Auth
}

type Auth interface {
	Login(
		ctx context.Context,
		email string,
		password string,
		appID int32,
	) (token string, refresh string, exp int64, err error)
	RegisterNewUser(
		ctx context.Context,
		name string,
		phone string,
		address string,
		email string,
		password string,
	) (userId int64, resName string, resEmail string, activated bool, err error)
	Logout(ctx context.Context,
		refresh string,
	) (success bool, err error)
	GetUserInfo(
		ctx context.Context,
		token string,
	) (userId int64, email string, name string, phone string, address string, activated bool, err error)
	RefreshToken(
		ctx context.Context,
		refresh string,
	) (token string, refreshNew string, exp int64, err error)
	ForgotPassword(
		ctx context.Context,
		email string,
		appID int32,
	) (success bool, message string, exp int64, err error)
	ResetPassword(
		ctx context.Context,
		token string,
		password string,
	) (success bool, message string, err error)
	SendVerificationEmail(
		ctx context.Context,
		userId int64,
		appID int32,
	) (success bool, message string, exp int64, err error)
	EmailVerify(
		ctx context.Context,
		token string,
	) (success bool, message string, activated bool, err error)
}

func (s *authServer) Login(
	ctx context.Context,
	in *ssov1.LoginRequest,
) (*ssov1.LoginResponse, error) {
	if in.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	if in.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	if in.GetAppId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "app_id is required")
	}

	token, refresh, exp, err := s.auth.Login(ctx, in.GetEmail(), in.GetPassword(), in.GetAppId())
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, "invalid email or password")
		}

		return nil, status.Error(codes.Internal, "failed to login")
	}

	return &ssov1.LoginResponse{AccessToken: token, RefreshToken: refresh, ExpiresAtUnix: exp}, nil
}

func (s *authServer) Register(
	ctx context.Context,
	in *ssov1.RegisterRequest,
) (*ssov1.RegisterResponse, error) {
	if in.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	if in.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	userId, name, email, activated, err := s.auth.RegisterNewUser(ctx, in.GetName(), in.GetPhone(), in.GetAddress(), in.GetEmail(), in.GetPassword())
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}
		if errors.Is(err, services.ErrInvalidEmail) {
			return nil, status.Error(codes.InvalidArgument, "invalid email")
		}
		if errors.Is(err, services.ErrInvalidPassword) {
			return nil, status.Error(codes.InvalidArgument, "invalid password")
		}

		return nil, status.Error(codes.Internal, "failed to register user")
	}

	return &ssov1.RegisterResponse{UserId: userId, Name: name, Email: email, Activated: activated}, nil
}

func (s *authServer) Logout(
	ctx context.Context,
	in *ssov1.LogoutRequest,
) (*ssov1.LogoutResponse, error) {
	if in.GetRefreshToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required or invalid")
	}

	success, err := s.auth.Logout(ctx, in.GetRefreshToken())
	if err != nil {
		if errors.Is(err, storage.ErrTokenNotFound) {
			return nil, status.Error(codes.NotFound, "refresh token not found")
		}

		return nil, status.Error(codes.Internal, "failed to logout")
	}

	return &ssov1.LogoutResponse{Success: success}, nil
}

func (s *authServer) GetUserInfo(
	ctx context.Context,
	in *ssov1.GetUserInfoRequest,
) (*ssov1.GetUserInfoResponse, error) {
	if in.GetAccessToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "access token is required or invalid")
	}

	userID, email, name, phone, address, activated, err := s.auth.GetUserInfo(ctx, in.GetAccessToken())
	if err != nil {
		if errors.Is(err, storage.ErrTokenNotFound) {
			return nil, status.Error(codes.NotFound, "access token not found")
		}

		return nil, status.Error(codes.Internal, "failed to get user info")
	}

	return &ssov1.GetUserInfoResponse{UserId: userID, Email: email, Name: name, Phone: phone, Address: address, Activated: activated}, nil
}

func (s *authServer) RefreshToken(
	ctx context.Context,
	in *ssov1.RefreshTokenRequest,
) (*ssov1.RefreshTokenResponse, error) {
	if in.GetRefreshToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required or invalid")
	}

	token, refresh, exp, err := s.auth.RefreshToken(ctx, in.GetRefreshToken())
	if err != nil {
		if errors.Is(err, storage.ErrTokenNotFound) {
			return nil, status.Error(codes.NotFound, "refresh token is not found")
		}

		return nil, status.Error(codes.Internal, "failed to retrieve refreshed tokens")
	}

	return &ssov1.RefreshTokenResponse{
		AccessToken:   token,
		RefreshToken:  refresh,
		ExpiresAtUnix: exp,
	}, nil
}

func (s *authServer) ForgotPassword(
	ctx context.Context,
	in *ssov1.ForgotPasswordRequest,
) (*ssov1.ForgotPasswordResponse, error) {
	if in.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if in.GetAppId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "app_id is required")
	}

	success, message, exp, err := s.auth.ForgotPassword(ctx, in.GetEmail(), in.GetAppId())

	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, "invalid email or password")
		}

		return nil, status.Error(codes.Internal, "failed to send forgot password email")
	}

	return &ssov1.ForgotPasswordResponse{Success: success, Message: message, ExpiresAtUnix: exp}, nil
}

func (s *authServer) ResetPassword(
	ctx context.Context,
	in *ssov1.ResetPasswordRequest,
) (*ssov1.ResetPasswordResponse, error) {
	if in.GetToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}
	if in.GetNewPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "new password is required")
	}

	success, message, err := s.auth.ResetPassword(ctx, in.GetToken(), in.GetNewPassword())

	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, "invalid token")
		}

		return nil, status.Error(codes.Internal, "failed to reset password")
	}

	return &ssov1.ResetPasswordResponse{Success: success, Message: message}, nil
}

func (s *authServer) SendEmailVerification(
	ctx context.Context,
	in *ssov1.SendEmailVerificationRequest,
) (*ssov1.SendEmailVerificationResponse, error) {
	if in.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if in.GetAppId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "app_id is required")
	}

	success, message, exp, err := s.auth.SendVerificationEmail(ctx, in.GetUserId(), in.GetAppId())

	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, "invalid user id")
		}

		return nil, status.Error(codes.Internal, "failed to send email verification")
	}

	return &ssov1.SendEmailVerificationResponse{Success: success, Message: message, ExpiresAtUnix: exp}, nil
}

func (s *authServer) EmailVerify(
	ctx context.Context,
	in *ssov1.EmailVerifyRequest,
) (*ssov1.EmailVerifyResponse, error) {
	if in.GetToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	success, message, userActivated, err := s.auth.EmailVerify(ctx, in.GetToken())

	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, "invalid token")
		}
		return nil, status.Error(codes.Internal, "failed to verify email")
	}
	return &ssov1.EmailVerifyResponse{Success: success, Message: message, Activated: userActivated}, nil
}
