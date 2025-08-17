package application

import "github.com/google/wire"

// ProviderSet is the wire provider set for the posts application layer
var ProviderSet = wire.NewSet(
	NewPostsService,
	NewPostsOwnershipChecker,
)
