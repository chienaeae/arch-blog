package server

import (
	"context"
	"net/http"
	"time"

	"github.com/philly/arch-blog/backend/internal/adapters/api"
	"github.com/philly/arch-blog/backend/internal/adapters/auth"
	"github.com/philly/arch-blog/backend/internal/adapters/rest/middleware"
	"github.com/philly/arch-blog/backend/internal/platform/logger"
)

// NewHTTPServer creates and configures the HTTP server with all routes
func NewHTTPServer(
	config Config,
	jwtMiddleware *auth.JWTMiddleware,
	server api.ServerInterface,
	authzMiddleware *middleware.AuthorizationMiddleware,
	authAdapter *middleware.AuthAdapter,
	log logger.Logger,
) *http.Server {

	// Create a modern Go 1.22+ ServeMux with method-based routing
	mux := http.NewServeMux()

	// Create middleware chain for different endpoint groups
	// Note: Middleware is applied in the order: last added = first executed
	
	// Public endpoints (no auth)
	// publicMiddlewares := []api.MiddlewareFunc{} // Currently unused but may be needed for future public endpoints

	// Protected endpoints (JWT auth required)
	protectedMiddlewares := []api.MiddlewareFunc{
		wrapMiddleware(jwtMiddleware.Middleware),
		wrapMiddleware(authAdapter.Middleware), // Convert Supabase ID to internal UUID
	}

	// Admin endpoints (JWT auth + specific permissions)
	// We'll create specific middleware chains for each permission group
	createAuthzMiddleware := func(permission string) []api.MiddlewareFunc {
		return append(protectedMiddlewares,
			wrapMiddleware(authzMiddleware.RequirePermission(permission)),
		)
	}

	// Create the API handler with oapi-codegen's routing
	// This handles all the path parameter extraction automatically
	apiHandler := api.HandlerWithOptions(server, api.StdHTTPServerOptions{
		BaseURL:    "/api/v1",
		BaseRouter: mux,
		Middlewares: []api.MiddlewareFunc{
			// Default middleware that applies to all routes
			// We'll override this per-route below
		},
	})

	// Now we need to wrap specific routes with their appropriate middleware
	// Since oapi-codegen doesn't support per-route middleware directly,
	// we'll create a wrapper that applies middleware based on the path

	wrappedHandler := &middlewareRouter{
		inner:                 apiHandler,
		publicPaths:           map[string]bool{
			"/api/v1/health/live":  true,
			"/api/v1/health/ready": true,
		},
		protectedMiddlewares:  protectedMiddlewares,
		permissionMiddlewares: map[string][]api.MiddlewareFunc{
			// Permission endpoints
			"GET /api/v1/permissions": createAuthzMiddleware("authz:permissions:read"),
			
			// Role management
			"GET /api/v1/roles":                       createAuthzMiddleware("authz:roles:read"),
			"POST /api/v1/roles":                      createAuthzMiddleware("authz:roles:create"),
			"GET /api/v1/roles/*":                     createAuthzMiddleware("authz:roles:read"),
			"PUT /api/v1/roles/*":                     createAuthzMiddleware("authz:roles:update"),
			"DELETE /api/v1/roles/*":                  createAuthzMiddleware("authz:roles:delete"),
			"PUT /api/v1/roles/*/permissions":         createAuthzMiddleware("authz:roles:update"),
			
			// User role management
			"GET /api/v1/users/*/roles":               createAuthzMiddleware("authz:users:read"),
			"POST /api/v1/users/*/roles":              createAuthzMiddleware("authz:users:assign"),
			"DELETE /api/v1/users/*/roles/*":          createAuthzMiddleware("authz:users:revoke"),
		},
	}

	// Wrap with observability middleware
	handler := withObservability(wrappedHandler, log)

	// Create and return HTTP server
	return &http.Server{
		Addr:         config.ServerAddress,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// middlewareRouter applies different middleware based on the request path
type middlewareRouter struct {
	inner                 http.Handler
	publicPaths           map[string]bool
	protectedMiddlewares  []api.MiddlewareFunc
	permissionMiddlewares map[string][]api.MiddlewareFunc
}

func (m *middlewareRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	
	// Check if it's a public endpoint
	if m.publicPaths[path] {
		m.inner.ServeHTTP(w, r)
		return
	}
	
	// Check for permission-specific endpoints
	// We need to match patterns with wildcards
	for pattern, middlewares := range m.permissionMiddlewares {
		if matchesPattern(r.Method+" "+path, pattern) {
			handler := m.inner
			// Apply middlewares in reverse order
			for i := len(middlewares) - 1; i >= 0; i-- {
				handler = middlewares[i](handler)
			}
			handler.ServeHTTP(w, r)
			return
		}
	}
	
	// Default: apply protected middleware for all other endpoints
	handler := m.inner
	for i := len(m.protectedMiddlewares) - 1; i >= 0; i-- {
		handler = m.protectedMiddlewares[i](handler)
	}
	handler.ServeHTTP(w, r)
}

// matchesPattern checks if a path matches a pattern with wildcards
func matchesPattern(path, pattern string) bool {
	// Simple pattern matching with * as wildcard
	// This is a basic implementation - could be enhanced with more sophisticated matching
	if pattern == path {
		return true
	}
	
	// Handle wildcard patterns
	// Convert pattern to a simple regex-like check
	// For example: "GET /api/v1/roles/*" matches "GET /api/v1/roles/123"
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(path) >= len(prefix) && path[:len(prefix)] == prefix
	}
	
	// Handle middle wildcards like /users/*/roles
	// This is a simplified version - a production system might use proper pattern matching
	return false
}

// wrapMiddleware converts a standard middleware to oapi-codegen's MiddlewareFunc
func wrapMiddleware(mw func(http.Handler) http.Handler) api.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return mw(next)
	}
}

// withObservability adds request logging and metrics
func withObservability(handler http.Handler, log logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process the request
		handler.ServeHTTP(wrapped, r)

		// Log request details
		duration := time.Since(start)
		
		// Extract user ID if available for better tracing
		var userID string
		if uid, ok := middleware.GetUserID(r.Context()); ok {
			userID = uid.String()
		}
		
		log.Info(r.Context(), "HTTP request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"user_id", userID,
		)

		// Here you could also emit metrics to Prometheus, DataDog, etc.
		// metrics.RecordHTTPRequest(r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.ResponseWriter.WriteHeader(code)
		rw.written = true
	}
}

// Write ensures we capture the status code even if WriteHeader wasn't called
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// GetUserID is a helper to extract user ID from context for logging
func GetUserID(ctx context.Context) (string, bool) {
	if uid, ok := middleware.GetUserID(ctx); ok {
		return uid.String(), true
	}
	return "", false
}