package application

import (
	"github.com/google/wire"
)

// ProviderSet is the wire provider set for authz application services
var ProviderSet = wire.NewSet(
	NewAuthzService,
)
