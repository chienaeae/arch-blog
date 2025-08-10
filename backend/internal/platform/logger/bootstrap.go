package logger

import (
	"context"
	"log"
	"os"
)

// BootstrapLogger is a simple logger used during application startup
// before the main configuration is loaded. It has zero dependencies.
type BootstrapLogger struct {
	logger *log.Logger
}

// NewBootstrapLogger creates a simple logger for bootstrap phase
func NewBootstrapLogger() *BootstrapLogger {
	return &BootstrapLogger{
		logger: log.New(os.Stdout, "[BOOTSTRAP] ", log.LstdFlags|log.Lshortfile),
	}
}

// Debug logs a message at debug level
func (b *BootstrapLogger) Debug(ctx context.Context, msg string, args ...any) {
	b.logger.Printf("DEBUG: %s %v", msg, args)
}

// Info logs a message at info level
func (b *BootstrapLogger) Info(ctx context.Context, msg string, args ...any) {
	b.logger.Printf("INFO: %s %v", msg, args)
}

// Warn logs a message at warn level
func (b *BootstrapLogger) Warn(ctx context.Context, msg string, args ...any) {
	b.logger.Printf("WARN: %s %v", msg, args)
}

// Error logs a message at error level
func (b *BootstrapLogger) Error(ctx context.Context, msg string, args ...any) {
	b.logger.Printf("ERROR: %s %v", msg, args)
}

// Ensure BootstrapLogger implements Logger interface
var _ Logger = (*BootstrapLogger)(nil)