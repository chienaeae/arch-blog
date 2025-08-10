package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents the assignment of a role to a user
// This is a simple struct for API responses
type UserRole struct {
	UserID    uuid.UUID
	RoleID    uuid.UUID
	Role      *Role
	GrantedAt time.Time
	GrantedBy uuid.UUID
}