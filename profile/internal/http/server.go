//go:generate go run generate.go

package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	profilev1 "github.com/m4rk1sov/protos/gen/go/profile"
	"github.com/swaggest/swgui"
	"github.com/swaggest/swgui/v5emb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log/slog"
	"net/http"
	"time"
)

type Server struct {
	httpServer *http.Server
	grpcAddr   string
	port       int
	log        *slog.Logger
}

func NewServer(grpcAddr string, port int, log *slog.Logger) *Server {
	return &Server{
		grpcAddr: grpcAddr,
		port:     port,
		log:      log,
	}
}

func (s *Server) Start() error {
	const op = "http.Server.Start"

	log := s.log.With(
		slog.String("op", op),
		slog.Int("port", s.port),
	)

	// gRPC-Gateway mux for API endpoints
	gwMux := runtime.NewServeMux()

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	err := profilev1.RegisterProfileServiceHandlerFromEndpoint(context.Background(), gwMux, s.grpcAddr, opts)
	if err != nil {
		return fmt.Errorf("failed to register auth handler: %w", err)
	}

	// Main mux for swagger UI and API endpoints
	mainMux := http.NewServeMux()

	// Swagger UI first
	s.setupSwaggerUI(mainMux)

	// API endpoints
	mainMux.Handle("/v1/", gwMux)

	handler := corsMiddleware(mainMux)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("localhost:%d", s.port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Info("http server started", slog.String("addr", s.httpServer.Addr))

	go func() {
		err := s.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(fmt.Errorf("failed to start HTTP server: %w", err))
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) setupSwaggerUI(mux *http.ServeMux) {
	swaggerHandler := v5emb.NewHandlerWithConfig(swgui.Config{
		Title:       "SSO API Documentation",
		SwaggerJSON: "/swagger/swagger.json",
		BasePath:    "/swagger/",
		ShowTopBar:  true,
	})

	mux.HandleFunc("/swagger/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Request for swagger.json received\n")
		w.Header().Set("Content-Type", "application/json")

		file, err := SwaggerAssets.Open("/swagger.json")
		if err != nil {
			fmt.Printf("Error opening swagger.json: %v\n", err)
			http.Error(w, fmt.Sprintf("Swagger spec not found: %v", err), http.StatusNotFound)
			return
		}
		defer func(file http.File) {
			err := file.Close()
			if err != nil {
				return
			}
		}(file)

		fmt.Printf("Successfully opened swagger file\n")
		http.ServeContent(w, r, "swagger.json", time.Time{}, file)
	})

	mux.Handle("/swagger/", swaggerHandler)
	mux.Handle("/swagger", http.RedirectHandler("/swagger/", http.StatusFound))
}
