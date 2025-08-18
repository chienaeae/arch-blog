package eventbus

import "github.com/google/wire"

// ProviderSet is the wire provider set for the event bus.
var ProviderSet = wire.NewSet(NewBus)
