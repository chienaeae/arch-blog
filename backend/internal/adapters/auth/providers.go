package auth

import "github.com/google/wire"

// ProviderSet is the wire provider set for auth middleware
var ProviderSet = wire.NewSet(
	NewJWTMiddleware,
)