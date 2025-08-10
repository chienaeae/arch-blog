package authz_adapter

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/philly/arch-blog/backend/internal/authz/application"
	"github.com/philly/arch-blog/backend/internal/authz/permission"
)

// Authorizer is an adapter that wraps the AuthzService to implement
// authorization interfaces required by other bounded contexts.
// This adapter allows other modules to define their own authorization
// interfaces (consumer-driven contracts) without depending on the authz module.
//
// Example usage:
// - The posts module defines its own ports.Authorizer interface
// - This adapter implements that interface by delegating to AuthzService
// - Wire injects this adapter into the posts module
//
// This pattern ensures complete decoupling between bounded contexts.
type Authorizer struct {
	authzService *application.AuthzService
}

// NewAuthorizer creates a new authorization adapter
func NewAuthorizer(authzService *application.AuthzService) *Authorizer {
	return &Authorizer{
		authzService: authzService,
	}
}

// Can checks if a user can perform an action on a resource
// This is a common method that most bounded contexts will need
// - For global permissions (e.g., "posts:create"), pass resourceID as nil
// - For resource-specific permissions (e.g., "posts:update:own"), pass the resourceID
func (a *Authorizer) Can(ctx context.Context, userID uuid.UUID, permissionID string, resourceID *uuid.UUID) (bool, error) {
	// If no resource ID is provided, check global permission
	if resourceID == nil {
		return a.authzService.HasPermission(ctx, userID, permissionID)
	}
	
	// Use the central permission registry to get structured data
	perm, exists := permission.FromID(permissionID)
	if !exists {
		// This permission string isn't even registered in the system
		return false, fmt.Errorf("invalid permission: %s", permissionID)
	}
	
	// Check permission with resource ownership
	// The AuthzService will handle the complex logic:
	// 1. Check if user has the ":any" version of the permission
	// 2. If not, check if user has the ":own" permission AND owns the resource
	return a.authzService.HasPermissionForResource(ctx, userID, permissionID, perm.Resource, *resourceID)
}

// CanAny checks if a user can perform any of the given actions
func (a *Authorizer) CanAny(ctx context.Context, userID uuid.UUID, permissions []string) (bool, error) {
	return a.authzService.HasAnyPermission(ctx, userID, permissions)
}

// CanAll checks if a user can perform all of the given actions
func (a *Authorizer) CanAll(ctx context.Context, userID uuid.UUID, permissions []string) (bool, error) {
	return a.authzService.HasAllPermissions(ctx, userID, permissions)
}

// HasRole checks if a user has a specific role
// Some bounded contexts might need role-based checks in addition to permission checks
func (a *Authorizer) HasRole(ctx context.Context, userID uuid.UUID, role string) (bool, error) {
	return a.authzService.HasRole(ctx, userID, role)
}

// GetUserPermissions retrieves all permission IDs for a user
// Useful for UI/frontend to show/hide features based on permissions
func (a *Authorizer) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return a.authzService.GetUserPermissions(ctx, userID)
}

// GetUserRoles retrieves all role names for a user
// Useful for UI/frontend to display user roles
func (a *Authorizer) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return a.authzService.GetUserRoles(ctx, userID)
}