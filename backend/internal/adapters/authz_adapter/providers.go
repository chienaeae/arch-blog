package authz_adapter

import "github.com/google/wire"

// ProviderSet is the wire provider set for the authorization adapter
var ProviderSet = wire.NewSet(
	NewAuthorizer,
)