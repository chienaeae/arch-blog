package server

import (
	"context"
	"fmt"
	"time"

	"backend/internal/platform/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectDatabase creates a new database connection pool and returns it with a cleanup function
func ConnectDatabase(ctx context.Context, config Config, log logger.Logger) (*pgxpool.Pool, func(), error) {
	log.Info(ctx, "connecting to database")

	// Parse config from URL and set pool defaults
	poolConfig, err := pgxpool.ParseConfig(config.DatabaseURL)
	if err != nil {
		log.Error(ctx, "failed to parse database URL", "error", err)
		return nil, nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Configure connection pool settings
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 5 * time.Minute
	poolConfig.MaxConnIdleTime = 1 * time.Minute

	log.Debug(ctx, "database pool configuration",
		"max_conns", poolConfig.MaxConns,
		"min_conns", poolConfig.MinConns,
		"max_conn_lifetime", poolConfig.MaxConnLifetime,
		"max_conn_idle_time", poolConfig.MaxConnIdleTime,
	)

	// Create the connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Error(ctx, "failed to create connection pool", "error", err)
		return nil, nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		log.Error(ctx, "failed to ping database", "error", err)
		return nil, nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info(ctx, "database connection established successfully")

	// Return the pool and a cleanup function
	cleanup := func() {
		log.Info(context.Background(), "closing database connection pool")
		pool.Close()
	}

	return pool, cleanup, nil
}
