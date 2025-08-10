package rest

import (
	"github.com/philly/arch-blog/backend/internal/adapters/api"
)

// Server combines all handlers to implement api.ServerInterface
type Server struct {
	*UserHandler
	*HealthHandler
}

// NewServer creates a new server that implements api.ServerInterface
func NewServer(userHandler *UserHandler, healthHandler *HealthHandler) api.ServerInterface {
	return &Server{
		UserHandler:   userHandler,
		HealthHandler: healthHandler,
	}
}

// Ensure Server implements api.ServerInterface
var _ api.ServerInterface = (*Server)(nil)

// The methods are already implemented by the embedded handlers:
// - GetLiveness (from HealthHandler)
// - GetReadiness (from HealthHandler)  
// - CreateUser (from UserHandler)
// - GetCurrentUser (from UserHandler)