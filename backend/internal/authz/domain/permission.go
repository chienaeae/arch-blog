package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Error definitions for permission operations
var (
	ErrInvalidPermissionID = errors.New("invalid permission ID format")
)

// Permission represents a permission in the authorization system
type Permission struct {
	ID          uuid.UUID
	Resource    string // e.g., "posts", "users", "comments"
	Action      string // e.g., "create", "read", "update", "delete"
	Scope       string // e.g., "own", "any", "self", or empty for no scope
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewPermission creates a new Permission domain object with structured fields
func NewPermission(resource, action, scope, description string) *Permission {
	now := time.Now()
	return &Permission{
		ID:          uuid.New(),
		Resource:    resource,
		Action:      action,
		Scope:       scope,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// NewPermissionFromID creates a Permission by parsing a permission ID string
func NewPermissionFromID(permissionID, description string) (*Permission, error) {
	resource, action, scope := ParsePermissionID(permissionID)
	if resource == "" || action == "" {
		return nil, fmt.Errorf("%w: %s", ErrInvalidPermissionID, permissionID)
	}
	
	return NewPermission(resource, action, scope, description), nil
}

// IDString returns the full permission ID string (derived from fields)
func (p *Permission) IDString() string {
	if p.Scope != "" {
		return fmt.Sprintf("%s:%s:%s", p.Resource, p.Action, p.Scope)
	}
	return fmt.Sprintf("%s:%s", p.Resource, p.Action)
}

// IsOwnershipBased returns true if this permission is ownership-scoped
func (p *Permission) IsOwnershipBased() bool {
	return p.Scope == "own" || p.Scope == "self"
}

// IsGlobal returns true if this permission applies to any resource
func (p *Permission) IsGlobal() bool {
	return p.Scope == "any"
}

// GetResource returns the resource name (simple field accessor)
func (p *Permission) GetResource() string {
	return p.Resource
}

// GetAction returns the action (simple field accessor)
func (p *Permission) GetAction() string {
	return p.Action
}

// GetScope returns the scope (simple field accessor)
func (p *Permission) GetScope() string {
	return p.Scope
}

// HasScope returns true if the permission has a scope qualifier
func (p *Permission) HasScope() bool {
	return p.Scope != ""
}

// Matches checks if this permission matches the given resource and action
func (p *Permission) Matches(resource, action string) bool {
	return p.Resource == resource && p.Action == action
}

// ParsePermissionID parses a permission ID string into its components
func ParsePermissionID(permissionID string) (resource, action, scope string) {
	parts := strings.Split(permissionID, ":")
	
	switch len(parts) {
	case 2:
		// Format: "resource:action"
		return parts[0], parts[1], ""
	case 3:
		// Format: "resource:action:scope"
		return parts[0], parts[1], parts[2]
	case 4:
		// Format: "resource:action:qualifier:scope" (e.g., "posts:read:draft:own")
		// Combine middle parts as the action
		return parts[0], parts[1] + ":" + parts[2], parts[3]
	default:
		// Invalid format
		return "", "", ""
	}
}