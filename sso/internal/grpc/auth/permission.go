package auth

import (
	"context"
	// codes for grpc clients to understand
	"google.golang.org/grpc/codes"
	// status of errors for understanding of grpc clients
	"google.golang.org/grpc/status"

	ssov1 "github.com/m4rk1sov/protos/gen/go/sso"
)

type permissionServer struct {
	ssov1.UnimplementedPermissionServer
	permission Permission
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
