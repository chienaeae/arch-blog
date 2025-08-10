//go:build wireinject
// +build wireinject

package server

import (
	"context"

	"github.com/google/wire"
	"github.com/philly/arch-blog/backend/internal/adapters/auth"
	"github.com/philly/arch-blog/backend/internal/adapters/postgres"
	"github.com/philly/arch-blog/backend/internal/adapters/rest"
	"github.com/philly/arch-blog/backend/internal/platform/logger"
	"github.com/philly/arch-blog/backend/internal/users/application"
)

// InitializeApp creates a fully configured App with all dependencies
func InitializeApp(ctx context.Context) (*App, func(), error) {
	wire.Build(
		// Bootstrap phase
		logger.NewBootstrapLogger,
		LoadConfig,
		
		// Logger configuration
		provideLoggerConfig,
		
		// Main logger
		logger.NewConfiguredLogger,
		wire.Bind(new(logger.Logger), new(*logger.SlogAdapter)),
		
		// Database
		ConnectDatabase,
		
		// Repository providers (includes interface binding)
		postgres.ProviderSet,
		
		// Application services
		application.ProviderSet,
		
		// REST handlers
		rest.ProviderSet,
		provideVersion, // Provide version string for HealthHandler
		
		// Auth middleware
		provideJWTMiddleware,
		
		// HTTP Server
		NewHTTPServer,
		
		// App
		NewApp,
	)
	
	return nil, nil, nil
}

// provideJWTMiddleware creates JWT middleware from config
func provideJWTMiddleware(ctx context.Context, config Config) (*auth.JWTMiddleware, error) {
	return auth.NewJWTMiddleware(ctx, config.JWKSEndpoint, config.JWTIssuer)
}

// provideVersion provides the application version
func provideVersion() string {
	return "1.0.0"
}

// provideLoggerConfig creates logger config from server config
func provideLoggerConfig(config Config) logger.Config {
	return logger.Config{
		Environment: config.Environment,
		LogLevel:    config.LogLevel,
	}
}