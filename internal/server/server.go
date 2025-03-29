package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/copyleftdev/goscry/internal/config"
	"github.com/copyleftdev/goscry/internal/tasks"
)

type Server struct {
	httpServer  *http.Server
	cfg         *config.Config
	taskManager *tasks.Manager
	logger      *log.Logger
}

func NewServer(cfg *config.Config, tm *tasks.Manager, logger *log.Logger) *Server {
	apiHandler := NewAPIHandler(tm, logger)
	router := chi.NewRouter()

	// --- Middleware Setup ---
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	// Use custom logger adapting stdlib logger or replace with structured logger middleware
	router.Use(RequestLogger(logger))
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second)) // Request timeout

	// CORS Configuration
	corsOptions := cors.Options{
		AllowedOrigins:   cfg.Security.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-API-Key"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true, // Be careful with this in production
		MaxAge:           300,  // Maximum value not ignored by any major browsers
	}
	router.Use(cors.Handler(corsOptions))

	// API Key Authentication Middleware (Simple Example)
	if cfg.Security.ApiKey != "" {
		router.Use(APIKeyAuth(cfg.Security.ApiKey))
	}

	// --- Route Definitions ---
	router.Route("/api/v1", func(r chi.Router) {
		r.Post("/tasks", apiHandler.HandleSubmitTask)
		r.Get("/tasks/{taskID}", apiHandler.HandleGetTaskStatus)
		r.Post("/tasks/{taskID}/2fa", apiHandler.HandleProvide2FACode)
		r.Post("/dom/ast", apiHandler.HandleGetDomAST)
	})

	// Health check endpoint
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	})

	// --- HTTP Server Configuration ---
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
		ErrorLog:     logger, // Use the same logger for server errors
	}

	return &Server{
		httpServer:  httpServer,
		cfg:         cfg,
		taskManager: tm,
		logger:      logger,
	}
}

func (s *Server) Start() error {
	s.logger.Printf("Starting GoScry server on %s", s.httpServer.Addr)
	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Println("Shutting down server...")
	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}
	s.logger.Println("Server gracefully stopped.")
	return nil
}

// --- Custom Middleware ---

// RequestLogger adapts stdlib logger for basic request logging
func RequestLogger(logger *log.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			defer func() {
				logger.Printf(
					"\"%s %s %s\" from %s - %d %dB in %v",
					r.Method, r.RequestURI, r.Proto, r.RemoteAddr,
					ww.Status(), ww.BytesWritten(), time.Since(start),
				)
			}()
			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}

// APIKeyAuth provides simple API Key authentication
func APIKeyAuth(validKey string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// Allow pre-flight OPTIONS requests without auth
			if r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				// Check Authorization header as Bearer token as alternative
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					apiKey = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			if apiKey == "" {
				http.Error(w, http.StatusText(http.StatusUnauthorized)+": API key required", http.StatusUnauthorized)
				return
			}
			if apiKey != validKey {
				http.Error(w, http.StatusText(http.StatusForbidden)+": Invalid API key", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
