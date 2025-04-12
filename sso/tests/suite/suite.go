package suite

import (
	ssov1 "github.com/m4rk1sov/auth/gen/go/sso"
	"sso/internal/config"
	"testing"
)

type Suite struct {
	*testing.T
	Cfg        *config.Config
	AuthClient ssov1.AuthClient
}
