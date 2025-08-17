package ports

import (
	"context"

	"github.com/google/uuid"
)

// Authorizer is an interface for checking permissions
// This is a driven port - the posts module depends on this capability
// but doesn't know how it's implemented
type Authorizer interface {
	Can(ctx context.Context, userID uuid.UUID, resource string, action string, resourceID *uuid.UUID) (bool, error)
}