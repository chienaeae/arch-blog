package authz_adapter

import (
	postsPorts "backend/internal/posts/ports"
	themesPorts "backend/internal/themes/ports"
	"github.com/google/wire"
)

// ProviderSet is the wire provider set for the authorization adapter
var ProviderSet = wire.NewSet(
	NewAuthzAdapter,
	// Bind the AuthzAdapter to both ports interfaces
	wire.Bind(new(postsPorts.Authorizer), new(*AuthzAdapter)),
	wire.Bind(new(themesPorts.Authorizer), new(*AuthzAdapter)),
)
