package ownership

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// DefaultRegistry is the default implementation of Registry
type DefaultRegistry struct {
	checkers map[string]Checker
	mu       sync.RWMutex
}

// NewRegistry creates a new ownership registry
func NewRegistry() *DefaultRegistry {
	return &DefaultRegistry{
		checkers: make(map[string]Checker),
	}
}

// RegisterChecker registers an ownership checker for a resource type
func (r *DefaultRegistry) RegisterChecker(resourceType string, checker Checker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkers[resourceType] = checker
}

// GetChecker retrieves the ownership checker for a resource type
func (r *DefaultRegistry) GetChecker(resourceType string) (Checker, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	checker, exists := r.checkers[resourceType]
	return checker, exists
}

// CheckOwnership checks ownership for any registered resource type
func (r *DefaultRegistry) CheckOwnership(ctx context.Context, userID uuid.UUID, resourceType string, resourceID uuid.UUID) (bool, error) {
	checker, exists := r.GetChecker(resourceType)
	if !exists {
		return false, fmt.Errorf("no ownership checker registered for resource type: %s", resourceType)
	}
	
	return checker.CheckOwnership(ctx, userID, resourceID)
}