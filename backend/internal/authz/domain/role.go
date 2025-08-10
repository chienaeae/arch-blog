package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Error definitions for role operations
var (
	ErrPermissionNil         = errors.New("permission cannot be nil")
	ErrPermissionExists      = errors.New("permission already exists in role")
	ErrPermissionNotFound    = errors.New("permission not found in role")
	ErrOnlyTemplateCanClone  = errors.New("only template roles can be cloned")
	ErrTemplateCannotAssign  = errors.New("template roles cannot be assigned to users")
	ErrSystemCannotDelete    = errors.New("system roles cannot be deleted")
)

// Role represents a role in the authorization system
type Role struct {
	ID          uuid.UUID
	Name        string
	Description string
	IsTemplate  bool // Template roles cannot be assigned to users directly
	IsSystem    bool // System roles cannot be deleted
	Permissions []*Permission
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewRole creates a new Role domain object
func NewRole(name, description string) *Role {
	now := time.Now()
	return &Role{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		IsTemplate:  false,
		IsSystem:    false,
		Permissions: make([]*Permission, 0),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// NewSystemRole creates a new system role that cannot be deleted
func NewSystemRole(name, description string) *Role {
	role := NewRole(name, description)
	role.IsSystem = true
	return role
}

// NewTemplateRole creates a new template role for creating custom roles
func NewTemplateRole(name, description string) *Role {
	role := NewRole(name, description)
	role.IsTemplate = true
	role.IsSystem = true // Templates are also system roles
	return role
}

// AddPermission adds a permission to the role
func (r *Role) AddPermission(permission *Permission) error {
	if permission == nil {
		return ErrPermissionNil
	}
	
	// Check if permission already exists
	for _, p := range r.Permissions {
		if p.ID == permission.ID {
			return ErrPermissionExists
		}
	}
	
	r.Permissions = append(r.Permissions, permission)
	r.UpdatedAt = time.Now()
	return nil
}

// RemovePermission removes a permission from the role
func (r *Role) RemovePermission(permissionID uuid.UUID) error {
	for i, p := range r.Permissions {
		if p.ID == permissionID {
			// Remove the permission by slicing
			r.Permissions = append(r.Permissions[:i], r.Permissions[i+1:]...)
			r.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrPermissionNotFound
}

// HasPermission checks if the role has a specific permission
func (r *Role) HasPermission(permissionID string) bool {
	for _, p := range r.Permissions {
		if p.IDString() == permissionID {
			return true
		}
	}
	return false
}

// HasPermissionForResource checks if the role has any permission for a resource
func (r *Role) HasPermissionForResource(resource, action string) bool {
	for _, p := range r.Permissions {
		if p.Matches(resource, action) {
			return true
		}
	}
	return false
}

// CanBeAssigned checks if this role can be assigned to a user
func (r *Role) CanBeAssigned() bool {
	return !r.IsTemplate
}

// CanBeDeleted checks if this role can be deleted
func (r *Role) CanBeDeleted() bool {
	return !r.IsSystem
}

// CloneAsCustomRole creates a new non-template role based on this template
func (r *Role) CloneAsCustomRole(newName, newDescription string) (*Role, error) {
	if !r.IsTemplate {
		return nil, ErrOnlyTemplateCanClone
	}
	
	newRole := NewRole(newName, newDescription)
	
	// Copy all permissions from the template
	newRole.Permissions = append(newRole.Permissions, r.Permissions...)
	
	return newRole, nil
}

// Validate checks if the role assignment is valid
func (r *Role) Validate() error {
	if r.IsTemplate {
		return ErrTemplateCannotAssign
	}
	return nil
}

// ValidateDeletion checks if the role can be deleted
func (r *Role) ValidateDeletion() error {
	if r.IsSystem {
		return ErrSystemCannotDelete
	}
	return nil
}