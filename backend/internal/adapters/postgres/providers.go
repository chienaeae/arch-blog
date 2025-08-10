package postgres

import (
	"github.com/google/wire"
	"github.com/philly/arch-blog/backend/internal/authz/ports"
)

// ProviderSet is the wire provider set for postgres repositories
var ProviderSet = wire.NewSet(
	NewUserRepository,
	// NewUserRepository already returns ports.UserRepository interface
	NewAuthzRepository,
	wire.Bind(new(ports.AuthzRepository), new(*AuthzRepository)),
)