package rest

import (
	"github.com/google/wire"
)

// ProviderSet is the wire provider set for REST handlers
var ProviderSet = wire.NewSet(
	NewBaseHandler,
	NewUserHandler,
	NewHealthHandler,
	NewAuthzHandler,
	NewPostsHandler,
	NewThemesHandler,
	NewServer, // Combined server that implements api.ServerInterface
)
