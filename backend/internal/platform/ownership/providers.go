package ownership

import "github.com/google/wire"

// ProviderSet is the wire provider set for ownership registry
var ProviderSet = wire.NewSet(
	NewRegistry,
	wire.Bind(new(Registry), new(*DefaultRegistry)),
)