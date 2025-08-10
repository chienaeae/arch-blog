package logger

import (
	"github.com/google/wire"
)

// ProviderSet is the wire provider set for the logger.
var ProviderSet = wire.NewSet(
	NewBootstrapLogger,
	NewConfiguredLogger,
	wire.Bind(new(Logger), new(*SlogAdapter)),
)

// Config holds the values needed to configure the logger
type Config struct {
	Environment string
	LogLevel    string
}

// NewConfiguredLogger creates the main application logger from config
func NewConfiguredLogger(config Config) *SlogAdapter {
	return NewSlogAdapter(config.Environment, config.LogLevel)
}