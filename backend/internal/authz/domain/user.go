package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Error definitions for user authorization operations
var (
	ErrRoleNil              = errors.New("role cannot be nil")
	ErrRoleAlreadyExists    = errors.New("user already has this role")
	ErrRoleNotFound         = errors.New("role not found for user")
	ErrCannotAssignTemplate = errors.New("cannot assign template role to user")
)

// UserAuthz represents a user's authorization information
// This domain object is optimized for COMMAND operations (mutations)
// For QUERY operations (permission checks), use the AuthzService directly
type UserAuthz struct {
	UserID            uuid.UUID
	Roles             []*Role
	CustomPermissions []*Permission // Direct permissions granted to the user
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// NewUserAuthz creates a new UserAuthz domain object
func NewUserAuthz(userID uuid.UUID) *UserAuthz {
	now := time.Now()
	return &UserAuthz{
		UserID:            userID,
		Roles:             make([]*Role, 0),
		CustomPermissions: make([]*Permission, 0),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// ===== COMMAND OPERATIONS =====
// These methods modify the user's authorization state

// AddRole assigns a role to the user
func (u *UserAuthz) AddRole(role *Role) error {
	if role == nil {
		return ErrRoleNil
	}

	// Validate that the role can be assigned
	if err := role.Validate(); err != nil {
		return err
	}

	// Check if role already exists
	for _, r := range u.Roles {
		if r.ID == role.ID {
			return ErrRoleAlreadyExists
		}
	}

	u.Roles = append(u.Roles, role)
	u.UpdatedAt = time.Now()
	return nil
}

// RemoveRole removes a role from the user
func (u *UserAuthz) RemoveRole(roleID uuid.UUID) error {
	for i, r := range u.Roles {
		if r.ID == roleID {
			// Remove the role by slicing
			u.Roles = append(u.Roles[:i], u.Roles[i+1:]...)
			u.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrRoleNotFound
}

// AddCustomPermission grants a direct permission to the user
func (u *UserAuthz) AddCustomPermission(permission *Permission) error {
	if permission == nil {
		return ErrPermissionNil
	}

	// Check if permission already exists
	for _, p := range u.CustomPermissions {
		if p.ID == permission.ID {
			return ErrPermissionExists
		}
	}

	u.CustomPermissions = append(u.CustomPermissions, permission)
	u.UpdatedAt = time.Now()
	return nil
}

// RemoveCustomPermission removes a direct permission from the user
func (u *UserAuthz) RemoveCustomPermission(permissionID uuid.UUID) error {
	for i, p := range u.CustomPermissions {
		if p.ID == permissionID {
			// Remove the permission by slicing
			u.CustomPermissions = append(u.CustomPermissions[:i], u.CustomPermissions[i+1:]...)
			u.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrPermissionNotFound
}

// ReplaceAllRoles replaces all user roles with a new set
func (u *UserAuthz) ReplaceAllRoles(roles []*Role) error {
	// Validate all roles first
	for _, role := range roles {
		if role == nil {
			return ErrRoleNil
		}
		if err := role.Validate(); err != nil {
			return err
		}
	}

	u.Roles = roles
	u.UpdatedAt = time.Now()
	return nil
}

// ClearAllRoles removes all roles from the user
func (u *UserAuthz) ClearAllRoles() {
	u.Roles = make([]*Role, 0)
	u.UpdatedAt = time.Now()
}

// ClearAllCustomPermissions removes all custom permissions from the user
func (u *UserAuthz) ClearAllCustomPermissions() {
	u.CustomPermissions = make([]*Permission, 0)
	u.UpdatedAt = time.Now()
}

// ===== QUERY OPERATIONS =====
// These methods are kept minimal and should only be used when the full object is already loaded
// For performance-critical permission checks, use AuthzService.HasPermission() instead

// GetRoleIDs returns the IDs of all roles assigned to the user
func (u *UserAuthz) GetRoleIDs() []uuid.UUID {
	ids := make([]uuid.UUID, len(u.Roles))
	for i, role := range u.Roles {
		ids[i] = role.ID
	}
	return ids
}

// GetCustomPermissionIDs returns the IDs of all custom permissions
func (u *UserAuthz) GetCustomPermissionIDs() []uuid.UUID {
	ids := make([]uuid.UUID, len(u.CustomPermissions))
	for i, perm := range u.CustomPermissions {
		ids[i] = perm.ID
	}
	return ids
}

// HasRoleID checks if the user has a specific role by ID (local check only)
func (u *UserAuthz) HasRoleID(roleID uuid.UUID) bool {
	for _, r := range u.Roles {
		if r.ID == roleID {
			return true
		}
	}
	return false
}

// CountRoles returns the number of roles assigned to the user
func (u *UserAuthz) CountRoles() int {
	return len(u.Roles)
}

// CountCustomPermissions returns the number of custom permissions
func (u *UserAuthz) CountCustomPermissions() int {
	return len(u.CustomPermissions)
}
