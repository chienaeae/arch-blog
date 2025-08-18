//go:build wireinject
// +build wireinject

package server

import (
	"context"

	"backend/internal/adapters/authz_adapter"
	"backend/internal/adapters/postgres"
	"backend/internal/adapters/rest"
	"backend/internal/adapters/rest/middleware"
	authzApp "backend/internal/authz/application"
	"backend/internal/platform/eventbus"
	"backend/internal/platform/logger"
	"backend/internal/platform/ownership"
	postgresDb "backend/internal/platform/postgres"
	postsApp "backend/internal/posts/application"
	themesApp "backend/internal/themes/application"
	"backend/internal/users/application"
	"github.com/google/wire"
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

		// Platform services
		postgresDb.NewTransactionManager,
		ownership.ProviderSet,
		eventbus.NewBus,

		// Repository providers (includes interface binding)
		postgres.ProviderSet,

		// Cross-context adapters
		authz_adapter.ProviderSet,

		// Application services
		application.ProviderSet,
		authzApp.ProviderSet,
		postsApp.ProviderSet,
		themesApp.ProviderSet,

		// REST handlers
		rest.ProviderSet,
		provideVersion, // Provide version string for HealthHandler

		// Auth middleware
		provideJWTConfig,
		middleware.ProviderSet,

		// HTTP Server
		NewHTTPServer,

		// App
		NewApp,
	)

	return nil, nil, nil
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

// provideJWTConfig adapts server Config into middleware.JWTConfig to avoid package cycles
func provideJWTConfig(config Config) middleware.JWTConfig {
	return middleware.JWTConfig{
		JWKS:   config.JWKSEndpoint,
		Issuer: config.JWTIssuer,
	}
}
