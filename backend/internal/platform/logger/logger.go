package logger

import (
	"context"
)

// Logger defines the interface for logging.
// This allows us to swap out the underlying logging implementation.
type Logger interface {
	Debug(ctx context.Context, msg string, args ...any)
	Info(ctx context.Context, msg string, args ...any)
	Warn(ctx context.Context, msg string, args ...any)
	Error(ctx context.Context, msg string, args ...any)
}
