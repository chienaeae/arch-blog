package server

import (
	"net/http"
	"time"

	"github.com/philly/arch-blog/backend/internal/adapters/auth"
	"github.com/philly/arch-blog/backend/internal/adapters/rest"
)

// NewHTTPServer creates and configures the HTTP server with all routes
func NewHTTPServer(
	config Config,
	jwtMiddleware *auth.JWTMiddleware,
	userHandler *rest.UserHandler,
	healthHandler *rest.HealthHandler,
) *http.Server {
	// Create the combined server that implements api.ServerInterface
	server := rest.NewServer(userHandler, healthHandler)

	// Create a modern Go 1.22+ ServeMux with method-based routing
	mux := http.NewServeMux()

	// Register PUBLIC health endpoints (no auth required)
	mux.HandleFunc("GET /api/v1/health/live", server.GetLiveness)
	mux.HandleFunc("GET /api/v1/health/ready", server.GetReadiness)

	// Register PROTECTED user endpoints (with JWT middleware)
	// Wrap handlers with authentication middleware
	createUserHandler := http.HandlerFunc(server.CreateUser)
	mux.Handle("POST /api/v1/users", jwtMiddleware.Middleware(createUserHandler))

	getCurrentUserHandler := http.HandlerFunc(server.GetCurrentUser)
	mux.Handle("GET /api/v1/users/me", jwtMiddleware.Middleware(getCurrentUserHandler))

	// Future protected endpoints can be added here:
	// createPostHandler := http.HandlerFunc(server.CreatePost)
	// mux.Handle("POST /api/v1/posts", jwtMiddleware.Middleware(createPostHandler))

	// Create and return HTTP server
	return &http.Server{
		Addr:         config.ServerAddress,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}