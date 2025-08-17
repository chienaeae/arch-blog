package application

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/philly/arch-blog/backend/internal/authz/domain"
	"github.com/philly/arch-blog/backend/internal/authz/permission"
	"github.com/philly/arch-blog/backend/internal/authz/ports"
	"github.com/philly/arch-blog/backend/internal/platform/apperror"
	"github.com/philly/arch-blog/backend/internal/platform/logger"
	"github.com/philly/arch-blog/backend/internal/platform/ownership"
)

// Error definitions for service operations using AppError
var (
	ErrUserNotFound = apperror.New(
		apperror.CodeNotFound,
		apperror.BusinessCodeUserNotFound,
		"user not found",
		http.StatusNotFound,
	)
	ErrRoleNotFound = apperror.New(
		apperror.CodeNotFound,
		apperror.BusinessCodeRoleNotFound,
		"role not found",
		http.StatusNotFound,
	)
	ErrPermissionNotFound = apperror.New(
		apperror.CodeNotFound,
		apperror.BusinessCodePermissionNotFound,
		"permission not found",
		http.StatusNotFound,
	)
	ErrUnauthorized = apperror.New(
		apperror.CodeUnauthorized,
		apperror.BusinessCodePermissionDenied,
		"unauthorized",
		http.StatusUnauthorized,
	)
	ErrInvalidPermission = apperror.New(
		apperror.CodeBadRequest,
		apperror.BusinessCodeInvalidPermission,
		"invalid permission",
		http.StatusBadRequest,
	)
	ErrResourceNotFound = apperror.New(
		apperror.CodeNotFound,
		apperror.BusinessCodeGeneral,
		"resource not found",
		http.StatusNotFound,
	)
	ErrRoleNameExists = apperror.New(
		apperror.CodeConflict,
		apperror.BusinessCodeRoleNameExists,
		"role name already exists",
		http.StatusConflict,
	)
	ErrRoleAlreadyAssigned = apperror.New(
		apperror.CodeConflict,
		apperror.BusinessCodeRoleAlreadyAssigned,
		"role already assigned to user",
		http.StatusConflict,
	)
	ErrRoleNotAssigned = apperror.New(
		apperror.CodeNotFound,
		apperror.BusinessCodeRoleNotAssigned,
		"role not assigned to user",
		http.StatusNotFound,
	)
	ErrTemplateCannotAssign = apperror.New(
		apperror.CodeBadRequest,
		apperror.BusinessCodeTemplateCannotAssign,
		"template roles cannot be assigned to users",
		http.StatusBadRequest,
	)
	ErrCannotUpdateSystemRole = apperror.New(
		apperror.CodeConflict,
		apperror.BusinessCodeCannotUpdateSystem,
		"cannot update system role",
		http.StatusConflict,
	)
	ErrCannotDeleteSystemRole = apperror.New(
		apperror.CodeConflict,
		apperror.BusinessCodeCannotDeleteSystem,
		"cannot delete system role",
		http.StatusConflict,
	)
)

// AuthzService implements the authorization business logic
type AuthzService struct {
	repo              ports.AuthzRepository
	ownershipRegistry ownership.Registry
	logger            logger.Logger
}

// NewAuthzService creates a new authorization service
func NewAuthzService(
	repo ports.AuthzRepository,
	ownershipRegistry ownership.Registry,
	logger logger.Logger,
) *AuthzService {
	return &AuthzService{
		repo:              repo,
		ownershipRegistry: ownershipRegistry,
		logger:            logger,
	}
}

// ===== QUERY OPERATIONS (Optimized for Performance) =====

// HasPermission checks if a user has a specific permission
// This is the primary method for authorization checks
func (s *AuthzService) HasPermission(ctx context.Context, userID uuid.UUID, permissionID string) (bool, error) {
	// Validate the permission ID exists in our constants
	if err := s.validatePermissionID(permissionID); err != nil {
		s.logger.Warn(ctx, "invalid permission requested",
			"user_id", userID,
			"permission", permissionID,
		)
		return false, err
	}

	// Use optimized repository query
	hasPermission, err := s.repo.HasPermission(ctx, userID, permissionID)
	if err != nil {
		s.logger.Error(ctx, "failed to check permission",
			"user_id", userID,
			"permission", permissionID,
			"error", err,
		)
		return false, fmt.Errorf("AuthzService.HasPermission: %w", err)
	}

	return hasPermission, nil
}

// HasPermissionForResource checks if a user has permission for a specific resource
// This handles ownership-based permissions (e.g., "posts:update:own")
func (s *AuthzService) HasPermissionForResource(
	ctx context.Context,
	userID uuid.UUID,
	permissionID string,
	resourceType string,
	resourceID uuid.UUID,
) (bool, error) {
	// First, check if the permission is valid
	perm, exists := permission.FromID(permissionID)
	if !exists {
		return false, fmt.Errorf("%w: %s", ErrInvalidPermission, permissionID)
	}

	// Optimization: Check for "any" permission first (most powerful)
	// This avoids expensive ownership checks for admin users
	if perm.Scope == "own" || perm.Scope == "self" {
		// Build the "any" version of this permission
		anyPermission := strings.Replace(permissionID, ":own", ":any", 1)
		anyPermission = strings.Replace(anyPermission, ":self", ":any", 1)
		
		// Check if user has the "any" version first
		hasAnyPermission, err := s.repo.HasPermission(ctx, userID, anyPermission)
		if err != nil {
			return false, fmt.Errorf("AuthzService.HasPermissionForResource (any check): %w", err)
		}
		if hasAnyPermission {
			return true, nil // User has global permission, no need to check ownership
		}

		// Now check ownership since they don't have the "any" permission
		isOwner, err := s.checkOwnership(ctx, userID, resourceType, resourceID)
		if err != nil {
			return false, fmt.Errorf("AuthzService.HasPermissionForResource (ownership check): %w", err)
		}
		if !isOwner {
			return false, nil // Not owner and doesn't have "any" permission
		}
		// User is owner, fall through to check the "own" permission
	}

	// Check the actual permission requested
	hasPermission, err := s.repo.HasPermission(ctx, userID, permissionID)
	if err != nil {
		return false, fmt.Errorf("AuthzService.HasPermissionForResource: %w", err)
	}
	
	return hasPermission, nil
}

// HasAnyPermission checks if a user has any of the specified permissions
func (s *AuthzService) HasAnyPermission(ctx context.Context, userID uuid.UUID, permissionIDs []string) (bool, error) {
	// Validate all permissions first
	if err := s.validatePermissionIDs(permissionIDs); err != nil {
		return false, err
	}

	hasAny, err := s.repo.HasAnyPermission(ctx, userID, permissionIDs)
	if err != nil {
		return false, fmt.Errorf("AuthzService.HasAnyPermission: %w", err)
	}
	
	return hasAny, nil
}

// Can is a simplified authorization check method that builds the permission ID
// from resource and action, then checks if the user has permission.
// This method is designed to be used by adapters that bridge to other modules.
// 
// For resource-specific checks (when resourceID is not nil), it will:
// 1. Check for "resource:action:any" permission (global permission)
// 2. If not found, check for "resource:action:own" with ownership verification
// 
// For non-resource checks (when resourceID is nil), it will:
// - Check for "resource:action" permission
func (s *AuthzService) Can(ctx context.Context, userID uuid.UUID, resource string, action string, resourceID *uuid.UUID) (bool, error) {
	// Build the base permission ID
	permissionID := fmt.Sprintf("%s:%s", resource, action)
	
	// If no resourceID, just check the basic permission
	if resourceID == nil {
		return s.HasPermission(ctx, userID, permissionID)
	}
	
	// For resource-specific checks, use HasPermissionForResource
	return s.HasPermissionForResource(ctx, userID, permissionID+":own", resource, *resourceID)
}

// HasAllPermissions checks if a user has all of the specified permissions
func (s *AuthzService) HasAllPermissions(ctx context.Context, userID uuid.UUID, permissionIDs []string) (bool, error) {
	// Validate all permissions first
	if err := s.validatePermissionIDs(permissionIDs); err != nil {
		return false, err
	}

	hasAll, err := s.repo.HasAllPermissions(ctx, userID, permissionIDs)
	if err != nil {
		return false, fmt.Errorf("AuthzService.HasAllPermissions: %w", err)
	}
	
	return hasAll, nil
}

// HasRole checks if a user has a specific role
func (s *AuthzService) HasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error) {
	hasRole, err := s.repo.HasRole(ctx, userID, roleName)
	if err != nil {
		return false, fmt.Errorf("AuthzService.HasRole: %w", err)
	}
	
	return hasRole, nil
}

// GetUserPermissions retrieves all permission IDs for a user
func (s *AuthzService) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	permissions, err := s.repo.GetUserPermissionIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("AuthzService.GetUserPermissions: %w", err)
	}
	
	return permissions, nil
}

// GetUserRoles retrieves all role names for a user
func (s *AuthzService) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	roles, err := s.repo.GetUserRoleNames(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("AuthzService.GetUserRoles: %w", err)
	}
	
	return roles, nil
}

// ===== COMMAND OPERATIONS (Modifications) =====

// AssignRoleToUser assigns a role to a user
func (s *AuthzService) AssignRoleToUser(ctx context.Context, userID, roleID, grantedBy uuid.UUID) error {
	// Get the role to validate it can be assigned
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("AuthzService.AssignRoleToUser (get role): %w", err)
	}

	// Validate the role can be assigned
	if err := role.Validate(); err != nil {
		return fmt.Errorf("AuthzService.AssignRoleToUser (validate): %w", err)
	}

	// Assign the role
	if err := s.repo.AssignRoleToUser(ctx, userID, roleID, grantedBy); err != nil {
		s.logger.Error(ctx, "failed to assign role to user",
			"user_id", userID,
			"role_id", roleID,
			"granted_by", grantedBy,
			"error", err,
		)
		return fmt.Errorf("AuthzService.AssignRoleToUser: %w", err)
	}

	s.logger.Info(ctx, "role assigned to user",
		"user_id", userID,
		"role_id", roleID,
		"role_name", role.Name,
		"granted_by", grantedBy,
	)

	return nil
}

// RemoveRoleFromUser removes a role from a user
func (s *AuthzService) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	if err := s.repo.RemoveRoleFromUser(ctx, userID, roleID); err != nil {
		s.logger.Error(ctx, "failed to remove role from user",
			"user_id", userID,
			"role_id", roleID,
			"error", err,
		)
		return fmt.Errorf("AuthzService.RemoveRoleFromUser: %w", err)
	}

	s.logger.Info(ctx, "role removed from user",
		"user_id", userID,
		"role_id", roleID,
	)

	return nil
}

// GrantPermissionToUser grants a custom permission to a user
func (s *AuthzService) GrantPermissionToUser(ctx context.Context, userID, permissionID, grantedBy uuid.UUID) error {
	// Verify the permission exists
	perm, err := s.repo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return fmt.Errorf("AuthzService.GrantPermissionToUser (get permission): %w", err)
	}

	// Grant the permission
	if err := s.repo.GrantPermissionToUser(ctx, userID, permissionID, grantedBy); err != nil {
		s.logger.Error(ctx, "failed to grant permission to user",
			"user_id", userID,
			"permission_id", permissionID,
			"granted_by", grantedBy,
			"error", err,
		)
		return fmt.Errorf("AuthzService.GrantPermissionToUser: %w", err)
	}

	s.logger.Info(ctx, "permission granted to user",
		"user_id", userID,
		"permission_id", permissionID,
		"permission_name", perm.IDString(),
		"granted_by", grantedBy,
	)

	return nil
}

// RevokePermissionFromUser revokes a custom permission from a user
func (s *AuthzService) RevokePermissionFromUser(ctx context.Context, userID, permissionID uuid.UUID) error {
	if err := s.repo.RevokePermissionFromUser(ctx, userID, permissionID); err != nil {
		s.logger.Error(ctx, "failed to revoke permission from user",
			"user_id", userID,
			"permission_id", permissionID,
			"error", err,
		)
		return fmt.Errorf("AuthzService.RevokePermissionFromUser: %w", err)
	}

	s.logger.Info(ctx, "permission revoked from user",
		"user_id", userID,
		"permission_id", permissionID,
	)

	return nil
}

// ReplaceUserRoles replaces all user roles atomically
func (s *AuthzService) ReplaceUserRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID, grantedBy uuid.UUID) error {
	// Validate all roles first
	for _, roleID := range roleIDs {
		role, err := s.repo.GetRoleByID(ctx, roleID)
		if err != nil {
			return fmt.Errorf("AuthzService.ReplaceUserRoles (get role %s): %w", roleID, err)
		}
		if err := role.Validate(); err != nil {
			return fmt.Errorf("AuthzService.ReplaceUserRoles (validate role %s): %w", roleID, err)
		}
	}

	// Replace the roles
	if err := s.repo.ReplaceUserRoles(ctx, userID, roleIDs, grantedBy); err != nil {
		s.logger.Error(ctx, "failed to replace user roles",
			"user_id", userID,
			"role_count", len(roleIDs),
			"granted_by", grantedBy,
			"error", err,
		)
		return fmt.Errorf("AuthzService.ReplaceUserRoles: %w", err)
	}

	s.logger.Info(ctx, "user roles replaced",
		"user_id", userID,
		"role_count", len(roleIDs),
		"granted_by", grantedBy,
	)

	return nil
}

// ===== EXTENDED QUERY OPERATIONS =====

// GetAllPermissions retrieves all permissions in the system
func (s *AuthzService) GetAllPermissions(ctx context.Context) ([]*domain.Permission, error) {
	permissions, err := s.repo.GetAllPermissions(ctx)
	if err != nil {
		s.logger.Error(ctx, "failed to get all permissions", "error", err)
		return nil, fmt.Errorf("AuthzService.GetAllPermissions: %w", err)
	}
	return permissions, nil
}

// GetAllRoles retrieves all roles in the system
func (s *AuthzService) GetAllRoles(ctx context.Context) ([]*domain.Role, error) {
	roles, err := s.repo.GetAllRoles(ctx)
	if err != nil {
		s.logger.Error(ctx, "failed to get all roles", "error", err)
		return nil, fmt.Errorf("AuthzService.GetAllRoles: %w", err)
	}
	return roles, nil
}

// GetRole retrieves a single role by ID
func (s *AuthzService) GetRole(ctx context.Context, roleID uuid.UUID) (*domain.Role, error) {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			return nil, err
		}
		s.logger.Error(ctx, "failed to get role", "role_id", roleID, "error", err)
		return nil, fmt.Errorf("AuthzService.GetRole: %w", err)
	}
	return role, nil
}

// GetUserRolesWithDetails retrieves all roles assigned to a user with full details
func (s *AuthzService) GetUserRolesWithDetails(ctx context.Context, userID uuid.UUID) ([]*domain.UserRole, error) {
	// Get user authorization data
	userAuthz, err := s.repo.GetUserAuthz(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
		s.logger.Error(ctx, "failed to get user roles", "user_id", userID, "error", err)
		return nil, fmt.Errorf("AuthzService.GetUserRolesWithDetails: %w", err)
	}

	// Map roles to UserRole objects
	userRoles := make([]*domain.UserRole, 0, len(userAuthz.Roles))
	for _, role := range userAuthz.Roles {
		userRole := &domain.UserRole{
			UserID:    userID,
			RoleID:    role.ID,
			Role:      role,
			GrantedAt: userAuthz.CreatedAt, // Using user creation time as fallback
			GrantedBy: uuid.Nil,            // We don't track this currently
		}
		userRoles = append(userRoles, userRole)
	}

	return userRoles, nil
}

// ===== ROLE MANAGEMENT =====

// CreateRole creates a new role
func (s *AuthzService) CreateRole(ctx context.Context, name, description string, isTemplate bool) (*domain.Role, error) {
	// Check if role name already exists
	existingRole, err := s.repo.GetRoleByName(ctx, name)
	if err == nil && existingRole != nil {
		return nil, ErrRoleNameExists
	}

	var role *domain.Role
	if isTemplate {
		role = domain.NewTemplateRole(name, description)
	} else {
		role = domain.NewRole(name, description)
	}

	if err := s.repo.CreateRole(ctx, role); err != nil {
		s.logger.Error(ctx, "failed to create role",
			"name", name,
			"is_template", isTemplate,
			"error", err,
		)
		return nil, fmt.Errorf("AuthzService.CreateRole: %w", err)
	}

	s.logger.Info(ctx, "role created",
		"role_id", role.ID,
		"name", name,
		"is_template", isTemplate,
	)

	return role, nil
}

// CreateRoleFromTemplate creates a new role based on a template
func (s *AuthzService) CreateRoleFromTemplate(ctx context.Context, templateID uuid.UUID, name, description string) (*domain.Role, error) {
	// Get the template role
	template, err := s.repo.GetRoleByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("AuthzService.CreateRoleFromTemplate (get template): %w", err)
	}

	// Clone the template
	newRole, err := template.CloneAsCustomRole(name, description)
	if err != nil {
		return nil, fmt.Errorf("AuthzService.CreateRoleFromTemplate (clone): %w", err)
	}

	// Save the new role
	if err := s.repo.CreateRole(ctx, newRole); err != nil {
		s.logger.Error(ctx, "failed to create role from template",
			"template_id", templateID,
			"name", name,
			"error", err,
		)
		return nil, fmt.Errorf("AuthzService.CreateRoleFromTemplate: %w", err)
	}

	s.logger.Info(ctx, "role created from template",
		"role_id", newRole.ID,
		"template_id", templateID,
		"name", name,
	)

	return newRole, nil
}

// UpdateRole updates a role's name and description
func (s *AuthzService) UpdateRole(ctx context.Context, roleID uuid.UUID, name, description *string) (*domain.Role, error) {
	// Get the existing role
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("AuthzService.UpdateRole (get role): %w", err)
	}

	// Check if the role can be updated
	if role.IsSystem {
		return nil, ErrCannotUpdateSystemRole
	}

	// Update fields if provided
	if name != nil && *name != "" {
		// Check if new name already exists (if different from current)
		if *name != role.Name {
			existingRole, err := s.repo.GetRoleByName(ctx, *name)
			if err == nil && existingRole != nil && existingRole.ID != roleID {
				return nil, ErrRoleNameExists
			}
		}
		role.Name = *name
	}

	if description != nil {
		role.Description = *description
	}

	// Update the role
	if err := s.repo.UpdateRole(ctx, role); err != nil {
		s.logger.Error(ctx, "failed to update role",
			"role_id", roleID,
			"error", err,
		)
		return nil, fmt.Errorf("AuthzService.UpdateRole: %w", err)
	}

	s.logger.Info(ctx, "role updated",
		"role_id", roleID,
		"name", role.Name,
	)

	return role, nil
}

// UpdateRolePermissions replaces all permissions for a role
func (s *AuthzService) UpdateRolePermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) (*domain.Role, error) {
	// Get the existing role
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("AuthzService.UpdateRolePermissions (get role): %w", err)
	}

	// Check if the role's permissions can be updated
	if role.IsSystem {
		return nil, ErrCannotUpdateSystemRole
	}

	// Verify all permissions exist
	for _, permID := range permissionIDs {
		_, err := s.repo.GetPermissionByID(ctx, permID)
		if err != nil {
			if errors.Is(err, ErrPermissionNotFound) {
				return nil, ErrPermissionNotFound
			}
			return nil, fmt.Errorf("AuthzService.UpdateRolePermissions (verify permission %s): %w", permID, err)
		}
	}

	// Update the permissions
	if err := s.repo.AssignPermissionsToRole(ctx, roleID, permissionIDs); err != nil {
		s.logger.Error(ctx, "failed to update role permissions",
			"role_id", roleID,
			"permission_count", len(permissionIDs),
			"error", err,
		)
		return nil, fmt.Errorf("AuthzService.UpdateRolePermissions: %w", err)
	}

	// Fetch the updated role
	updatedRole, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("AuthzService.UpdateRolePermissions (get updated role): %w", err)
	}

	s.logger.Info(ctx, "role permissions updated",
		"role_id", roleID,
		"permission_count", len(permissionIDs),
	)

	return updatedRole, nil
}

// DeleteRole deletes a role
func (s *AuthzService) DeleteRole(ctx context.Context, roleID uuid.UUID) error {
	// Get the role to validate it can be deleted
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("AuthzService.DeleteRole (get role): %w", err)
	}

	// Validate the role can be deleted
	if err := role.ValidateDeletion(); err != nil {
		return fmt.Errorf("AuthzService.DeleteRole (validate): %w", err)
	}

	// Delete the role
	if err := s.repo.DeleteRole(ctx, roleID); err != nil {
		s.logger.Error(ctx, "failed to delete role",
			"role_id", roleID,
			"error", err,
		)
		return fmt.Errorf("AuthzService.DeleteRole: %w", err)
	}

	s.logger.Info(ctx, "role deleted",
		"role_id", roleID,
		"name", role.Name,
	)

	return nil
}

// ===== PRIVATE HELPER METHODS =====

// validatePermissionID validates a single permission ID
func (s *AuthzService) validatePermissionID(permissionID string) error {
	if !permission.IsValid(permissionID) {
		return fmt.Errorf("%w: %s", ErrInvalidPermission, permissionID)
	}
	return nil
}

// validatePermissionIDs validates multiple permission IDs
func (s *AuthzService) validatePermissionIDs(permissionIDs []string) error {
	for _, permID := range permissionIDs {
		if err := s.validatePermissionID(permID); err != nil {
			return err
		}
	}
	return nil
}

// checkOwnership checks if a user owns a resource
func (s *AuthzService) checkOwnership(ctx context.Context, userID uuid.UUID, resourceType string, resourceID uuid.UUID) (bool, error) {
	if s.ownershipRegistry == nil {
		s.logger.Warn(ctx, "ownership registry not configured",
			"resource_type", resourceType,
		)
		return false, nil
	}

	isOwner, err := s.ownershipRegistry.CheckOwnership(ctx, userID, resourceType, resourceID)
	if err != nil {
		return false, fmt.Errorf("checkOwnership: %w", err)
	}
	
	return isOwner, nil
}