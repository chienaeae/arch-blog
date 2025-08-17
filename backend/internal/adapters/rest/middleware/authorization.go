package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/philly/arch-blog/backend/internal/authz/application"
	"github.com/philly/arch-blog/backend/internal/platform/logger"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// UserIDKey is the context key for the authenticated user's ID
	UserIDKey contextKey = "userID"
	
	// ResourceIDKey is the context key for the resource ID in ownership checks
	ResourceIDKey contextKey = "resourceID"

	// UserEmailKey is the context key for the authenticated user's email
	UserEmailKey contextKey = "userEmail"
)

// AuthorizationMiddleware provides permission-based authorization for HTTP handlers
type AuthorizationMiddleware struct {
	authzService *application.AuthzService
	logger       logger.Logger
}

// NewAuthorizationMiddleware creates a new authorization middleware
func NewAuthorizationMiddleware(authzService *application.AuthzService, logger logger.Logger) *AuthorizationMiddleware {
	return &AuthorizationMiddleware{
		authzService: authzService,
		logger:       logger,
	}
}

// RequirePermission creates a middleware that checks if the user has a specific permission
func (m *AuthorizationMiddleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			
			// Get user ID from context (should be set by authentication middleware)
			userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
			if !ok {
				m.logger.Warn(ctx, "user ID not found in context")
				WriteJSONError(w, ErrorCodeUnauthorized, "Authentication required", http.StatusUnauthorized)
				return
			}
			
			// Check permission
			hasPermission, err := m.authzService.HasPermission(ctx, userID, permission)
			if err != nil {
				m.logger.Error(ctx, "failed to check permission",
					"user_id", userID,
					"permission", permission,
					"error", err,
				)
				WriteJSONError(w, ErrorCodeInternalServerError, "Failed to check permissions", http.StatusInternalServerError)
				return
			}
			
			if !hasPermission {
				m.logger.Warn(ctx, "permission denied",
					"user_id", userID,
					"permission", permission,
				)
				WriteJSONError(w, ErrorCodeForbidden, "Insufficient permissions", http.StatusForbidden)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission creates a middleware that checks if the user has any of the specified permissions
func (m *AuthorizationMiddleware) RequireAnyPermission(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			
			// Get user ID from context
			userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
			if !ok {
				m.logger.Warn(ctx, "user ID not found in context")
				WriteJSONError(w, ErrorCodeUnauthorized, "Authentication required", http.StatusUnauthorized)
				return
			}
			
			// Check permissions
			hasPermission, err := m.authzService.HasAnyPermission(ctx, userID, permissions)
			if err != nil {
				m.logger.Error(ctx, "failed to check permissions",
					"user_id", userID,
					"permissions", permissions,
					"error", err,
				)
				WriteJSONError(w, ErrorCodeInternalServerError, "Failed to check permissions", http.StatusInternalServerError)
				return
			}
			
			if !hasPermission {
				m.logger.Warn(ctx, "permission denied",
					"user_id", userID,
					"required_permissions", permissions,
				)
				WriteJSONError(w, ErrorCodeForbidden, "Insufficient permissions", http.StatusForbidden)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAllPermissions creates a middleware that checks if the user has all of the specified permissions
func (m *AuthorizationMiddleware) RequireAllPermissions(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			
			// Get user ID from context
			userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
			if !ok {
				m.logger.Warn(ctx, "user ID not found in context")
				WriteJSONError(w, ErrorCodeUnauthorized, "Authentication required", http.StatusUnauthorized)
				return
			}
			
			// Check permissions
			hasPermissions, err := m.authzService.HasAllPermissions(ctx, userID, permissions)
			if err != nil {
				m.logger.Error(ctx, "failed to check permissions",
					"user_id", userID,
					"permissions", permissions,
					"error", err,
				)
				WriteJSONError(w, ErrorCodeInternalServerError, "Failed to check permissions", http.StatusInternalServerError)
				return
			}
			
			if !hasPermissions {
				m.logger.Warn(ctx, "permission denied",
					"user_id", userID,
					"required_permissions", permissions,
				)
				WriteJSONError(w, ErrorCodeForbidden, "Insufficient permissions", http.StatusForbidden)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole creates a middleware that checks if the user has a specific role
func (m *AuthorizationMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			
			// Get user ID from context
			userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
			if !ok {
				m.logger.Warn(ctx, "user ID not found in context")
				WriteJSONError(w, ErrorCodeUnauthorized, "Authentication required", http.StatusUnauthorized)
				return
			}
			
			// Check role
			hasRole, err := m.authzService.HasRole(ctx, userID, role)
			if err != nil {
				m.logger.Error(ctx, "failed to check role",
					"user_id", userID,
					"role", role,
					"error", err,
				)
				WriteJSONError(w, ErrorCodeInternalServerError, "Failed to check permissions", http.StatusInternalServerError)
				return
			}
			
			if !hasRole {
				m.logger.Warn(ctx, "role denied",
					"user_id", userID,
					"role", role,
				)
				WriteJSONError(w, ErrorCodeForbidden, "Insufficient permissions", http.StatusForbidden)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// RequireResourcePermission creates a middleware that checks ownership-based permissions
// It expects the resource ID to be in the URL parameters (e.g., /posts/{id})
func (m *AuthorizationMiddleware) RequireResourcePermission(permission, resourceType, urlParam string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			
			// Get user ID from context
			userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
			if !ok {
				m.logger.Warn(ctx, "user ID not found in context")
				WriteJSONError(w, ErrorCodeUnauthorized, "Authentication required", http.StatusUnauthorized)
				return
			}
			
			// Get resource ID from URL
			resourceIDStr := r.PathValue(urlParam)
			if resourceIDStr == "" {
				m.logger.Warn(ctx, "resource ID not found in URL",
					"param", urlParam,
				)
				WriteJSONError(w, ErrorCodeValidationError, "Invalid request parameters", http.StatusBadRequest)
				return
			}
			
			resourceID, err := uuid.Parse(resourceIDStr)
			if err != nil {
				m.logger.Warn(ctx, "invalid resource ID",
					"resource_id", resourceIDStr,
					"error", err,
				)
				WriteJSONError(w, ErrorCodeValidationError, "Invalid request parameters", http.StatusBadRequest)
				return
			}
			
			// Check permission with ownership
			hasPermission, err := m.authzService.HasPermissionForResource(
				ctx, userID, permission, resourceType, resourceID,
			)
			if err != nil {
				m.logger.Error(ctx, "failed to check resource permission",
					"user_id", userID,
					"permission", permission,
					"resource_type", resourceType,
					"resource_id", resourceID,
					"error", err,
				)
				WriteJSONError(w, ErrorCodeInternalServerError, "Failed to check permissions", http.StatusInternalServerError)
				return
			}
			
			if !hasPermission {
				m.logger.Warn(ctx, "resource permission denied",
					"user_id", userID,
					"permission", permission,
					"resource_type", resourceType,
					"resource_id", resourceID,
				)
				WriteJSONError(w, ErrorCodeForbidden, "Insufficient permissions", http.StatusForbidden)
				return
			}
			
			// Add resource ID to context for handlers
			ctx = context.WithValue(ctx, ResourceIDKey, resourceID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID is a helper function to get the user ID from the request context
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
	return userID, ok
}

// GetResourceID is a helper function to get the resource ID from the request context
func GetResourceID(ctx context.Context) (uuid.UUID, bool) {
	resourceID, ok := ctx.Value(ResourceIDKey).(uuid.UUID)
	return resourceID, ok
}

// SetUserID is a helper function to set the user ID in the request context
// This should be called by the authentication middleware after validating the JWT
func SetUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// RequireOwnership creates a middleware that only allows access to resource owners
// This is a convenience method for common ownership-based permissions
func (m *AuthorizationMiddleware) RequireOwnership(resourceType, urlParam, action string) func(http.Handler) http.Handler {
	// Construct the ownership permission (e.g., "posts:update:own")
	permission := resourceType + ":" + action + ":own"
	return m.RequireResourcePermission(permission, resourceType, urlParam)
}