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
	NewServer, // Combined server that implements api.ServerInterface
)