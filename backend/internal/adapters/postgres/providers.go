package postgres

import (
	authzPorts "backend/internal/authz/ports"
	postsPorts "backend/internal/posts/ports"
	themesPorts "backend/internal/themes/ports"
	"github.com/google/wire"
)

// ProviderSet is the wire provider set for postgres repositories
var ProviderSet = wire.NewSet(
	NewUserRepository,
	// NewUserRepository already returns ports.UserRepository interface
	NewAuthzRepository,
	wire.Bind(new(authzPorts.AuthzRepository), new(*AuthzRepository)),
	NewPostRepository,
	wire.Bind(new(postsPorts.PostRepository), new(*PostRepository)),
	NewThemeRepository,
	wire.Bind(new(themesPorts.ThemeRepository), new(*ThemeRepository)),
)
