package rest

import (
	"github.com/philly/arch-blog/backend/internal/adapters/api"
)

// Server combines all handlers to implement api.ServerInterface
type Server struct {
	*UserHandler
	*HealthHandler
	*AuthzHandler
}

// NewServer creates a new server that implements api.ServerInterface
func NewServer(
	userHandler *UserHandler,
	healthHandler *HealthHandler,
	authzHandler *AuthzHandler,
) api.ServerInterface {
	return &Server{
		UserHandler:   userHandler,
		HealthHandler: healthHandler,
		AuthzHandler:  authzHandler,
	}
}

// Ensure Server implements api.ServerInterface
var _ api.ServerInterface = (*Server)(nil)

// The methods are already implemented by the embedded handlers:
// - GetLiveness, GetReadiness (from HealthHandler)
// - CreateUser, GetCurrentUser (from UserHandler)
// - ListPermissions, ListRoles, CreateRole, GetRole, UpdateRole, DeleteRole,
//   UpdateRolePermissions, GetUserRoles, AssignRoleToUser, RevokeRoleFromUser (from AuthzHandler)