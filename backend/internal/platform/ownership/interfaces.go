package ownership

import (
	"context"

	"github.com/google/uuid"
)

// Checker defines the interface for checking resource ownership
// This should be implemented by each bounded context that has ownership-based resources
type Checker interface {
	// CheckOwnership verifies if a user owns a specific resource
	CheckOwnership(ctx context.Context, userID uuid.UUID, resourceID uuid.UUID) (bool, error)
}

// Registry holds ownership checkers for different resource types
// This is used by the AuthzService to verify ownership-based permissions
type Registry interface {
	// RegisterChecker registers an ownership checker for a resource type
	RegisterChecker(resourceType string, checker Checker)

	// GetChecker retrieves the ownership checker for a resource type
	GetChecker(resourceType string) (Checker, bool)

	// CheckOwnership checks ownership for any registered resource type
	CheckOwnership(ctx context.Context, userID uuid.UUID, resourceType string, resourceID uuid.UUID) (bool, error)
}
