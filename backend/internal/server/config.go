package server

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"backend/internal/platform/logger"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL   string `mapstructure:"DATABASE_URL"`
	JWKSEndpoint  string `mapstructure:"JWKS_ENDPOINT"` // Generic JWKS endpoint for JWT validation
	JWTIssuer     string `mapstructure:"JWT_ISSUER"`    // Expected JWT issuer for validation
	ServerAddress string `mapstructure:"SERVER_ADDRESS"`
	Environment   string `mapstructure:"ENVIRONMENT"`
	LogLevel      string `mapstructure:"LOG_LEVEL"` // Logging level (debug, info, warn, error)
}

func LoadConfig(bootstrapLogger *logger.BootstrapLogger) (Config, error) {
	ctx := context.Background()

	// Load .env file if it exists (godotenv will find it automatically)
	// It's okay if the file doesn't exist - we'll use environment variables
	if err := godotenv.Load(); err != nil {
		bootstrapLogger.Info(ctx, "no .env file found, using environment variables only")
	} else {
		bootstrapLogger.Info(ctx, "loaded .env file")
	}

	// Create a new Viper instance
	v := viper.New()

	// Set default values
	v.SetDefault("DATABASE_URL", "postgresql://localhost:5432/archblog?sslmode=disable")
	v.SetDefault("SERVER_ADDRESS", ":8080")
	v.SetDefault("ENVIRONMENT", "development")
	v.SetDefault("LOG_LEVEL", "info")

	// Enable automatic environment variable reading
	// Viper will now see all environment variables, including those loaded by godotenv
	v.AutomaticEnv()

	// Replace dots with underscores in environment variable names
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Unmarshal the configuration into our struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		bootstrapLogger.Error(ctx, "failed to unmarshal configuration", "error", err)
		return Config{}, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	bootstrapLogger.Info(ctx, "configuration loaded",
		"environment", config.Environment,
		"log_level", config.LogLevel,
		"server_address", config.ServerAddress,
	)

	// Validate required configuration
	if config.JWKSEndpoint == "" {
		err := errors.New("JWKS_ENDPOINT is required")
		bootstrapLogger.Error(ctx, "configuration validation failed", "error", err)
		return Config{}, err
	}
	if config.JWTIssuer == "" {
		err := errors.New("JWT_ISSUER is required")
		bootstrapLogger.Error(ctx, "configuration validation failed", "error", err)
		return Config{}, err
	}

	bootstrapLogger.Info(ctx, "configuration validated successfully")
	return config, nil
}
