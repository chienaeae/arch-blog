package ports

import (
	"context"

	"backend/internal/authz/domain"
	"github.com/google/uuid"
)

// AuthzRepository defines the interface for authorization data persistence
// It follows CQRS principles: separate methods for commands (mutations) and queries
type AuthzRepository interface {
	// ===== PERMISSION OPERATIONS =====

	// GetPermissionByID retrieves a permission by its UUID
	GetPermissionByID(ctx context.Context, id uuid.UUID) (*domain.Permission, error)

	// GetPermissionByIDString retrieves a permission by its string identifier (e.g., "posts:create")
	GetPermissionByIDString(ctx context.Context, permissionID string) (*domain.Permission, error)

	// GetAllPermissions retrieves all permissions in the system
	GetAllPermissions(ctx context.Context) ([]*domain.Permission, error)

	// CreatePermission creates a new permission
	CreatePermission(ctx context.Context, permission *domain.Permission) error

	// UpdatePermission updates an existing permission
	UpdatePermission(ctx context.Context, permission *domain.Permission) error

	// DeletePermission deletes a permission by ID
	DeletePermission(ctx context.Context, id uuid.UUID) error

	// ===== ROLE OPERATIONS =====

	// GetRoleByID retrieves a role by its UUID (includes permissions)
	GetRoleByID(ctx context.Context, id uuid.UUID) (*domain.Role, error)

	// GetRoleByName retrieves a role by its name (includes permissions)
	GetRoleByName(ctx context.Context, name string) (*domain.Role, error)

	// GetAllRoles retrieves all roles in the system
	GetAllRoles(ctx context.Context) ([]*domain.Role, error)

	// GetRoleTemplates retrieves only template roles
	GetRoleTemplates(ctx context.Context) ([]*domain.Role, error)

	// CreateRole creates a new role
	CreateRole(ctx context.Context, role *domain.Role) error

	// UpdateRole updates an existing role
	UpdateRole(ctx context.Context, role *domain.Role) error

	// DeleteRole deletes a role by ID
	DeleteRole(ctx context.Context, id uuid.UUID) error

	// AssignPermissionsToRole assigns permissions to a role (replaces existing)
	AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error

	// AddPermissionToRole adds a single permission to a role
	AddPermissionToRole(ctx context.Context, roleID uuid.UUID, permissionID uuid.UUID) error

	// RemovePermissionFromRole removes a single permission from a role
	RemovePermissionFromRole(ctx context.Context, roleID uuid.UUID, permissionID uuid.UUID) error

	// ===== USER AUTHORIZATION OPERATIONS =====

	// GetUserAuthz retrieves full authorization data for a user (for commands)
	GetUserAuthz(ctx context.Context, userID uuid.UUID) (*domain.UserAuthz, error)

	// AssignRoleToUser assigns a role to a user
	AssignRoleToUser(ctx context.Context, userID uuid.UUID, roleID uuid.UUID, grantedBy uuid.UUID) error

	// RemoveRoleFromUser removes a role from a user
	RemoveRoleFromUser(ctx context.Context, userID uuid.UUID, roleID uuid.UUID) error

	// GrantPermissionToUser grants a custom permission to a user
	GrantPermissionToUser(ctx context.Context, userID uuid.UUID, permissionID uuid.UUID, grantedBy uuid.UUID) error

	// RevokePermissionFromUser revokes a custom permission from a user
	RevokePermissionFromUser(ctx context.Context, userID uuid.UUID, permissionID uuid.UUID) error

	// ReplaceUserRoles replaces all user roles atomically
	// Pass an empty slice to clear all roles
	ReplaceUserRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID, grantedBy uuid.UUID) error

	// ClearUserPermissions removes all custom permissions from a user
	ClearUserPermissions(ctx context.Context, userID uuid.UUID) error

	// ===== OPTIMIZED QUERY OPERATIONS =====
	// These methods are optimized for performance-critical authorization checks

	// HasPermission checks if a user has a specific permission (direct query)
	// This performs an optimized database query without loading full objects
	HasPermission(ctx context.Context, userID uuid.UUID, permissionID string) (bool, error)

	// HasAnyPermission checks if a user has any of the specified permissions
	HasAnyPermission(ctx context.Context, userID uuid.UUID, permissionIDs []string) (bool, error)

	// HasAllPermissions checks if a user has all of the specified permissions
	HasAllPermissions(ctx context.Context, userID uuid.UUID, permissionIDs []string) (bool, error)

	// HasRole checks if a user has a specific role (direct query)
	HasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error)

	// GetUserPermissionIDs gets all permission IDs for a user (optimized)
	// Returns a list of permission_id strings without loading full objects
	GetUserPermissionIDs(ctx context.Context, userID uuid.UUID) ([]string, error)

	// GetUserRoleNames gets all role names for a user (optimized)
	GetUserRoleNames(ctx context.Context, userID uuid.UUID) ([]string, error)
}
