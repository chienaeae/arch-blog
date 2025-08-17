package postgres

import (
	"github.com/google/wire"
	authzPorts "github.com/philly/arch-blog/backend/internal/authz/ports"
	postsPorts "github.com/philly/arch-blog/backend/internal/posts/ports"
	themesPorts "github.com/philly/arch-blog/backend/internal/themes/ports"
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