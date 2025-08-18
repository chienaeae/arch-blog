package authz_adapter

import (
	"context"

	authzApp "backend/internal/authz/application"
	postsPorts "backend/internal/posts/ports"
	themesPorts "backend/internal/themes/ports"
	"github.com/google/uuid"
)

// AuthzAdapter is the unified adapter that bridges the authz service
// with all bounded contexts that need authorization.
// It implements multiple Authorizer interfaces from different modules.
type AuthzAdapter struct {
	authzService *authzApp.AuthzService
}

// NewAuthzAdapter creates a new authorization adapter
func NewAuthzAdapter(authzService *authzApp.AuthzService) *AuthzAdapter {
	return &AuthzAdapter{
		authzService: authzService,
	}
}

// Can checks if a user has permission to perform an action on a resource
// This method satisfies multiple interfaces:
// - themes/ports.Authorizer
// - posts/ports.Authorizer
// - any other module's Authorizer interface
func (a *AuthzAdapter) Can(ctx context.Context, userID uuid.UUID, resource string, action string, resourceID *uuid.UUID) (bool, error) {
	return a.authzService.Can(ctx, userID, resource, action, resourceID)
}

// Compile-time checks to ensure we implement the interfaces
var (
	_ postsPorts.Authorizer  = (*AuthzAdapter)(nil)
	_ themesPorts.Authorizer = (*AuthzAdapter)(nil)
)
