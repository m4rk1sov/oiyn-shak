package auth

import (
	"context"
	"errors"
	"sso/internal/services/auth"
	"sso/internal/storage"
	// codes for grpc clients to understand
	"google.golang.org/grpc/codes"
	// status of errors for understanding of grpc clients
	"google.golang.org/grpc/status"

	"google.golang.org/grpc"

	ssov1 "github.com/m4rk1sov/protos/gen/go/sso"
)

type authServer struct {
	// to resolve compatibility issues, when adding new proto generated files
	// will return not implemented error
	ssov1.UnimplementedAuthServer
	auth Auth
}

type permissionServer struct {
	ssov1.UnimplementedPermissionServer
	permission Permission
}

type Auth interface {
	Login(
		ctx context.Context,
		email string,
		password string,
		appID int,
	) (token string, refresh string, exp int64, err error)
	RegisterNewUser(
		ctx context.Context,
		email string,
		password string,
	) (userId int64, err error)
	Logout(ctx context.Context,
		refresh string,
	) (success bool, err error)
	GetUserInfo(
		ctx context.Context,
		token string,
	) (userId int64, email string, err error)
	RefreshToken(
		ctx context.Context,
		refresh string,
	) (token string, refreshNew string, exp int64, err error)
}

type Permission interface {
	GetUserPermissions(
		ctx context.Context,
		userId int64,
	) (permissions []string, err error)
	HasUserPermission(
		ctx context.Context,
		userId int64,
		permission string,
	) (allowed bool, err error)
}

func Register(gRPCServer *grpc.Server, auth Auth, permission Permission) {
	ssov1.RegisterAuthServer(gRPCServer, &authServer{auth: auth})
	ssov1.RegisterPermissionServer(gRPCServer, &permissionServer{permission: permission})
}

func (s *authServer) Login(
	ctx context.Context,
	in *ssov1.LoginRequest,
) (*ssov1.LoginResponse, error) {
	if in.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	if in.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	if in.GetAppId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "app_id is required")
	}

	token, refresh, exp, err := s.auth.Login(ctx, in.GetEmail(), in.GetPassword(), int(in.GetAppId()))
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
	if in.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	if in.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	userId, err := s.auth.RegisterNewUser(ctx, in.GetEmail(), in.GetPassword())
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}

		return nil, status.Error(codes.Internal, "failed to register user")
	}

	return &ssov1.RegisterResponse{UserId: userId}, nil
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

	userID, email, err := s.auth.GetUserInfo(ctx, in.GetAccessToken())
	if err != nil {
		if errors.Is(err, storage.ErrTokenNotFound) {
			return nil, status.Error(codes.NotFound, "access token not found")
		}

		return nil, status.Error(codes.Internal, "failed to get user info")
	}

	return &ssov1.GetUserInfoResponse{UserId: userID, Email: email}, nil
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

func (s *permissionServer) GetUserPermissions(
	ctx context.Context,
	in *ssov1.GetUserPermissionsRequest,
) (*ssov1.GetUserPermissionsResponse, error) {
	if in.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	permissions, err := s.permission.GetUserPermissions(ctx, in.GetUserId())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get the user permissions")
	}
	return &ssov1.GetUserPermissionsResponse{Permissions: permissions}, nil
}

func (s *permissionServer) HasUserPermission(
	ctx context.Context,
	in *ssov1.HasUserPermissionRequest,
) (*ssov1.HasUserPermissionResponse, error) {
	if in.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	if in.GetPermission() == "" {
		return nil, status.Error(codes.InvalidArgument, "permission is required")
	}

	allowed, err := s.permission.HasUserPermission(ctx, in.GetUserId(), in.GetPermission())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check user permission")
	}

	return &ssov1.HasUserPermissionResponse{Allowed: allowed}, nil
}
