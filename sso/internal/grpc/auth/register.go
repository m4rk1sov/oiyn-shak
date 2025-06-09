package auth

import (
	"google.golang.org/grpc"

	ssov1 "github.com/m4rk1sov/protos/gen/go/sso"
)

func Register(gRPCServer *grpc.Server, auth Auth, permission Permission) {
	ssov1.RegisterAuthServer(gRPCServer, &authServer{auth: auth})
	ssov1.RegisterPermissionServer(gRPCServer, &permissionServer{permission: permission})
}
