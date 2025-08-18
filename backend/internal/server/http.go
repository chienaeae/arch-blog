package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/philly/arch-blog/backend/internal/adapters/api"
	"github.com/philly/arch-blog/backend/internal/adapters/rest/middleware"
	"github.com/philly/arch-blog/backend/internal/platform/logger"
)

// NewHTTPServer creates and configures the HTTP server with all routes
func NewHTTPServer(
	config Config,
	server api.ServerInterface,
	jwtMiddleware *middleware.JWTMiddleware,
	authzMiddleware *middleware.AuthorizationMiddleware,
	authAdapter *middleware.AuthAdapter,
	log logger.Logger,
) *http.Server {
	// Create chi router
	r := chi.NewRouter()

	// Protected endpoints (JWT auth required)
	protectedMiddlewares := []api.MiddlewareFunc{
		wrapMiddleware(jwtMiddleware.Middleware),
		wrapMiddleware(authAdapter.Middleware), // Convert Supabase ID to internal UUID
	}

	// JWT-only endpoints (no AuthAdapter because user doesn't exist yet)
	jwtOnlyMiddlewares := []api.MiddlewareFunc{
		wrapMiddleware(jwtMiddleware.Middleware),
	}

	// Admin endpoints (JWT auth + specific permissions)
	// We'll create specific middleware chains for each permission group
	createAuthzMiddleware := func(permission string) []api.MiddlewareFunc {
		return append(protectedMiddlewares,
			wrapMiddleware(authzMiddleware.RequirePermission(permission)),
		)
	}

	// Ownership-based endpoints (JWT auth + ownership check)
	// For endpoints that require the user to own the resource
	createOwnershipMiddleware := func(resource string, urlParam string, action string) []api.MiddlewareFunc {
		return append(protectedMiddlewares,
			wrapMiddleware(authzMiddleware.RequireOwnership(resource, urlParam, action)),
		)
	}

	// Build route pattern maps for chi
	publicPatterns := map[string]bool{
		"GET /api/v1/health/live":  true,
		"GET /api/v1/health/ready": true,

		// Public posts endpoints (read-only)
		"GET /api/v1/posts":             true,
		"GET /api/v1/posts/{id}":        true, // Get by ID
		"GET /api/v1/posts/slug/{slug}": true, // Get by slug

		// Public themes endpoints (read-only)
		"GET /api/v1/themes":               true,
		"GET /api/v1/themes/{id}":          true, // Get by ID
		"GET /api/v1/themes/slug/{slug}":   true, // Get by slug
		"GET /api/v1/themes/{id}/articles": true, // Get theme with articles
	}

	permissionPatterns := map[string][]api.MiddlewareFunc{
		// User creation (JWT only, no AuthAdapter since user doesn't exist yet)
		"POST /api/v1/users": jwtOnlyMiddlewares,

		// Permission endpoints
		"GET /api/v1/permissions": createAuthzMiddleware("authz:permissions:read"),

		// Role management
		"GET /api/v1/roles":                  createAuthzMiddleware("authz:roles:read"),
		"POST /api/v1/roles":                 createAuthzMiddleware("authz:roles:create"),
		"GET /api/v1/roles/{id}":             createAuthzMiddleware("authz:roles:read"),
		"PUT /api/v1/roles/{id}":             createAuthzMiddleware("authz:roles:update"),
		"DELETE /api/v1/roles/{id}":          createAuthzMiddleware("authz:roles:delete"),
		"PUT /api/v1/roles/{id}/permissions": createAuthzMiddleware("authz:roles:update"),

		// User role management
		"GET /api/v1/users/{id}/roles":             createAuthzMiddleware("authz:users:read"),
		"POST /api/v1/users/{id}/roles":            createAuthzMiddleware("authz:users:assign"),
		"DELETE /api/v1/users/{id}/roles/{roleId}": createAuthzMiddleware("authz:users:revoke"),

		// Posts endpoints (mutation requires authorization)
		"POST /api/v1/posts":                createAuthzMiddleware("posts:create"),
		"PUT /api/v1/posts/{id}":            createOwnershipMiddleware("posts", "id", "update"),
		"POST /api/v1/posts/{id}/publish":   createOwnershipMiddleware("posts", "id", "publish"),
		"POST /api/v1/posts/{id}/unpublish": createOwnershipMiddleware("posts", "id", "publish"),
		"POST /api/v1/posts/{id}/archive":   createOwnershipMiddleware("posts", "id", "archive"),
		"DELETE /api/v1/posts/{id}":         createOwnershipMiddleware("posts", "id", "delete"),

		// Themes endpoints (mutation requires authorization)
		"POST /api/v1/themes":                          createAuthzMiddleware("themes:create"),
		"PUT /api/v1/themes/{id}":                      createOwnershipMiddleware("themes", "id", "update"),
		"POST /api/v1/themes/{id}/activate":            createOwnershipMiddleware("themes", "id", "update"),
		"POST /api/v1/themes/{id}/deactivate":          createOwnershipMiddleware("themes", "id", "update"),
		"POST /api/v1/themes/{id}/articles":            createOwnershipMiddleware("themes", "id", "update"),
		"DELETE /api/v1/themes/{id}/articles/{postId}": createOwnershipMiddleware("themes", "id", "update"),
		"PUT /api/v1/themes/{id}/articles":             createOwnershipMiddleware("themes", "id", "update"),
	}

	// Register API routes on chi router with a route-aware middleware
	_ = api.HandlerWithOptions(server, api.ChiServerOptions{
		BaseURL:    "/api/v1",
		BaseRouter: r,
		Middlewares: []api.MiddlewareFunc{
			routeAwareChiMiddleware(publicPatterns, permissionPatterns, protectedMiddlewares),
		},
	})
	// Wrap with observability middleware
	handler := withObservability(r, log)

	// Create and return HTTP server
	return &http.Server{
		Addr:         config.ServerAddress,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// routeAwareChiMiddleware applies auth middlewares based on matched chi route pattern
func routeAwareChiMiddleware(
	public map[string]bool,
	specific map[string][]api.MiddlewareFunc,
	defaults []api.MiddlewareFunc,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// chi exposes the current route pattern via RouteContext
			routeCtx := chi.RouteContext(r.Context())
			method := r.Method
			if method == "HEAD" {
				method = "GET"
			}
			pattern := ""
			if routeCtx != nil {
				pattern = method + " " + routeCtx.RoutePattern()
			}

			// Public endpoints bypass
			if public[pattern] || public[method+" "+r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Permission/ownership specific endpoints
			if middlewares, ok := specific[pattern]; ok {
				handler := next
				for i := len(middlewares) - 1; i >= 0; i-- {
					handler = middlewares[i](handler)
				}
				handler.ServeHTTP(w, r)
				return
			}

			// Default protected endpoints
			handler := next
			for i := len(defaults) - 1; i >= 0; i-- {
				handler = defaults[i](handler)
			}
			handler.ServeHTTP(w, r)
		})
	}
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

		// Use chi's response writer wrapper to capture status code and bytes written
		wrr := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

		// Process the request
		handler.ServeHTTP(wrr, r)

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
			"status", wrr.Status(),
			"duration_ms", duration.Milliseconds(),
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"user_id", userID,
		)

		// Here you could also emit metrics to Prometheus, DataDog, etc.
		// metrics.RecordHTTPRequest(r.Method, r.URL.Path, wrr.Status(), duration)
	})
}
