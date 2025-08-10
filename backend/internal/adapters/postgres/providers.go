package postgres

import "github.com/google/wire"

// ProviderSet is the wire provider set for postgres repositories
var ProviderSet = wire.NewSet(
	NewUserRepository,
	// NewUserRepository already returns ports.UserRepository interface
)