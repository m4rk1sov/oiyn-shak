package suite

import (
	"context"
	ssov1 "github.com/m4rk1sov/auth/gen/go/sso"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"os"
	"sso/internal/config"
	"strconv"
	"testing"
)

type Suite struct {
	*testing.T
	Cfg        *config.Config
	AuthClient ssov1.AuthClient
}

const (
	grpcHost = "localhost"
)

// New test suite
func New(t *testing.T) (context.Context, *Suite) {
	//err := godotenv.Load("../.env")
	//if err != nil {
	//	t.Fatal("Error loading .env file")
	//}

	t.Helper()   // helper for tests
	t.Parallel() // parallel test run

	cfg := config.MustLoadPath(configPath())

	// parent context
	ctx, cancelCtx := context.WithTimeout(context.Background(), cfg.GRPC.Timeout)

	// cleanup after tests are done (closing the context)
	t.Cleanup(func() {
		t.Helper()
		cancelCtx()
	})

	// client
	cc, err := grpc.DialContext(context.Background(),
		grpcAddress(cfg),
		// insecure connect for test
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc server connection failed: %v", err)
	}

	// gRPC client for server
	authClient := ssov1.NewAuthClient(cc)

	return ctx, &Suite{
		T:          t,
		Cfg:        cfg,
		AuthClient: authClient,
	}
}

func configPath() string {
	const key = "TEST_CONFIG_PATH"

	if v := os.Getenv(key); v != "" {
		return v
	}

	return "../config/local_tests_config.yaml"
}

func grpcAddress(cfg *config.Config) string {
	return net.JoinHostPort(grpcHost, strconv.Itoa(cfg.GRPC.Port))
}
