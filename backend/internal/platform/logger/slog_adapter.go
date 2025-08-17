package logger

import (
	"context"
	"log/slog"
	"os"
)

// SlogAdapter implements the Logger interface using Go's standard slog library.
type SlogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter creates a new logger based on the application configuration.
func NewSlogAdapter(env string, level string) *SlogAdapter {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	var handler slog.Handler
	if env == "development" {
		// Use a more human-readable text handler for development.
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	} else {
		// Use JSON handler for production, which is better for machine parsing.
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	}

	return &SlogAdapter{
		logger: slog.New(handler),
	}
}

// Debug logs a message at debug level
func (s *SlogAdapter) Debug(ctx context.Context, msg string, args ...any) {
	s.logger.DebugContext(ctx, msg, args...)
}

// Info logs a message at info level
func (s *SlogAdapter) Info(ctx context.Context, msg string, args ...any) {
	s.logger.InfoContext(ctx, msg, args...)
}

// Warn logs a message at warn level
func (s *SlogAdapter) Warn(ctx context.Context, msg string, args ...any) {
	s.logger.WarnContext(ctx, msg, args...)
}

// Error logs a message at error level
func (s *SlogAdapter) Error(ctx context.Context, msg string, args ...any) {
	s.logger.ErrorContext(ctx, msg, args...)
}
