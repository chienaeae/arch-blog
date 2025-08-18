package rest

import (
	"backend/internal/adapters/api"
)

// Server combines all handlers to implement api.ServerInterface
type Server struct {
	*UserHandler
	*HealthHandler
	*AuthzHandler
	*PostsHandler
	*ThemesHandler
}

// NewServer creates a new server that implements api.ServerInterface
func NewServer(
	userHandler *UserHandler,
	healthHandler *HealthHandler,
	authzHandler *AuthzHandler,
	postsHandler *PostsHandler,
	themesHandler *ThemesHandler,
) api.ServerInterface {
	return &Server{
		UserHandler:   userHandler,
		HealthHandler: healthHandler,
		AuthzHandler:  authzHandler,
		PostsHandler:  postsHandler,
		ThemesHandler: themesHandler,
	}
}

// Ensure Server implements api.ServerInterface
var _ api.ServerInterface = (*Server)(nil)
