//go:generate go run generate.go

package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	ssov1 "github.com/m4rk1sov/protos/gen/go/sso"
	"github.com/swaggest/swgui"
	"github.com/swaggest/swgui/v5emb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
	"time"
)

type Server struct {
	httpServer *http.Server
	grpcAddr   string
	port       int
	//swaggerSpec []byte
}

func NewServer(grpcAddr string, port int) *Server {
	return &Server{
		grpcAddr: grpcAddr,
		port:     port,
		//swaggerSpec: swaggerSpec,
	}
}

func (s *Server) Start() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// gRPC-Gateway mux for API endpoints
	gwMux := runtime.NewServeMux()

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	err := ssov1.RegisterAuthHandlerFromEndpoint(ctx, gwMux, s.grpcAddr, opts)
	if err != nil {
		return fmt.Errorf("failed to register auth handler: %w", err)
	}

	err = ssov1.RegisterPermissionHandlerFromEndpoint(ctx, gwMux, s.grpcAddr, opts)
	if err != nil {
		return fmt.Errorf("failed to register permission handler: %w", err)
	}

	// Main mux for swagger UI and API endpoints
	mainMux := http.NewServeMux()

	// Swagger UI first
	s.setupSwaggerUI(mainMux)

	// API endpoints
	mainMux.Handle("/", gwMux)
	//mainMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	//	if strings.HasPrefix(r.URL.Path, "/swagger") {
	//		http.NotFound(w, r)
	//		return
	//	}
	//	gwMux.ServeHTTP(w, r)
	//})

	handler := corsMiddleware(mainMux)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("localhost:%d", s.port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	//go func() {
	//	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
	//		cancel()
	//	}
	//}()
	//
	//<-ctx.Done()

	go func() {
		err := s.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(fmt.Errorf("failed to start HTTP server: %w", err))
		}
	}()

	return err
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

//func ServeSwagger(swaggerDir string) http.Handler {
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		if r.URL.Path == "/swagger" || r.URL.Path == "/swagger/" {
//			http.Redirect(w, r, "/swagger/index.html", http.StatusFound)
//			return
//		}
//
//		filePath := path.Join(swaggerDir, r.URL.Path[len("/swagger/"):])
//		http.ServeFile(w, r, filePath)
//	})
//}

func (s *Server) setupSwaggerUI(mux *http.ServeMux) {
	swaggerHandler := v5emb.NewHandlerWithConfig(swgui.Config{
		Title:       "SSO API Documentation",
		SwaggerJSON: "/swagger/swagger.json",
		BasePath:    "/swagger/",
		ShowTopBar:  true,
	})

	mux.HandleFunc("/swagger/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		//if len(s.swaggerSpec) > 0 {
		//	_, err := w.Write(s.swaggerSpec)
		//	if err != nil {
		//		return
		//	}
		//} else {
		file, err := SwaggerAssets.Open("swagger/swagger.json")
		if err != nil {
			http.Error(w, "Swagger spec not found", http.StatusNotFound)
			return
		}
		defer func(file http.File) {
			err := file.Close()
			if err != nil {
				return
			}
		}(file)
		http.ServeContent(w, r, "swagger.json", time.Time{}, file)
		//}
	})

	mux.Handle("/swagger/", swaggerHandler)
	mux.Handle("/swagger", http.RedirectHandler("/swagger/", http.StatusFound))
}
