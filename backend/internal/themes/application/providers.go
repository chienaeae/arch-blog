package application

import "github.com/google/wire"

// ProviderSet is the wire provider set for the themes application layer
var ProviderSet = wire.NewSet(
	NewThemesService,
	NewThemesOwnershipChecker,
	NewPostAdapter,
	wire.Bind(new(PostProvider), new(*PostAdapter)),
)
