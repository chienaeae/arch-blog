package application

import "github.com/google/wire"

// ProviderSet is the wire provider set for application services
var ProviderSet = wire.NewSet(
	NewUserService,
)