package authz_adapter

import (
	"github.com/google/wire"
	postsPorts "github.com/philly/arch-blog/backend/internal/posts/ports"
	themesPorts "github.com/philly/arch-blog/backend/internal/themes/ports"
)

// ProviderSet is the wire provider set for the authorization adapter
var ProviderSet = wire.NewSet(
	NewAuthzAdapter,
	// Bind the AuthzAdapter to both ports interfaces
	wire.Bind(new(postsPorts.Authorizer), new(*AuthzAdapter)),
	wire.Bind(new(themesPorts.Authorizer), new(*AuthzAdapter)),
)