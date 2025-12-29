package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/turnerem/zenzen/logger"
	"github.com/turnerem/zenzen/service"
)

type Server struct {
	store   service.Store
	router  *chi.Mux
	apiKey  string
	cognito *CognitoConfig
}

// NewServer creates a new API server
func NewServer(store service.Store, apiKey string) *Server {
	s := &Server{
		store:   store,
		router:  chi.NewRouter(),
		apiKey:  apiKey,
		cognito: nil,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// SetCognitoConfig sets the Cognito configuration for JWT authentication
func (s *Server) SetCognitoConfig(cognito *CognitoConfig) {
	s.cognito = cognito
}

func (s *Server) setupMiddleware() {
	// Basic middleware
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(60 * time.Second))

	// CORS configuration for mobile access
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Configure this properly for production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// API key authentication
	s.router.Use(s.authMiddleware)
}

func (s *Server) setupRoutes() {
	s.router.Get("/health", s.handleHealth)

	// API v1 routes
	s.router.Route("/api/v1", func(r chi.Router) {
		r.Get("/entries", s.handleGetEntries)
		r.Get("/entries/{id}", s.handleGetEntry)

		// Future: write endpoints
		// r.Post("/entries", s.handleCreateEntry)
		// r.Put("/entries/{id}", s.handleUpdateEntry)
		// r.Delete("/entries/{id}", s.handleDeleteEntry)
	})
}

// authMiddleware validates API key or Cognito JWT token
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Try Cognito JWT token first (if configured)
		if s.cognito != nil {
			bearerToken := extractBearerToken(r)
			if bearerToken != "" {
				_, err := s.cognito.ValidateToken(bearerToken)
				if err != nil {
					logger.Warn("cognito_token_validation_failed", "error", err.Error())
					http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
					return
				}

				// Token is valid
				logger.Info("authenticated", "method", "cognito")
				next.ServeHTTP(w, r)
				return
			}
		}

		// Fall back to API key authentication
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("api_key")
		}

		if apiKey != s.apiKey {
			logger.Warn("authentication_failed", "reason", "invalid_api_key")
			http.Error(w, "Unauthorized: Invalid or missing API key/token", http.StatusUnauthorized)
			return
		}

		logger.Info("authenticated", "method", "api_key")
		next.ServeHTTP(w, r)
	})
}

// Start starts the API server
func (s *Server) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	logger.Info("api_server_started", "address", addr)
	return http.ListenAndServe(addr, s.router)
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	// Chi doesn't have built-in server management
	// This would be implemented with http.Server
	return nil
}
