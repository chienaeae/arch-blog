package seeder

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/philly/arch-blog/backend/internal/platform/logger"
)

// Seeder defines the interface for all data seeders
type Seeder interface {
	// Name returns the name of the seeder for logging
	Name() string

	// Seed runs the seeding logic with database access
	// It should be idempotent - safe to run multiple times
	Seed(ctx context.Context, db *pgxpool.Pool) error
}

// Orchestrator manages and runs multiple seeders in order
type Orchestrator struct {
	seeders []Seeder
	logger  logger.Logger
	db      *pgxpool.Pool
}

// NewOrchestrator creates a new seeder orchestrator with all seeders injected
func NewOrchestrator(logger logger.Logger, db *pgxpool.Pool, seeders []Seeder) *Orchestrator {
	return &Orchestrator{
		seeders: seeders,
		logger:  logger,
		db:      db,
	}
}

// RunAll executes all registered seeders in order
func (o *Orchestrator) RunAll(ctx context.Context) error {
	o.logger.Info(ctx, "starting data seeding", "seeder_count", len(o.seeders))

	for _, seeder := range o.seeders {
		o.logger.Info(ctx, "running seeder", "seeder", seeder.Name())

		if err := seeder.Seed(ctx, o.db); err != nil {
			o.logger.Error(ctx, "seeder failed",
				"seeder", seeder.Name(),
				"error", err,
			)
			return fmt.Errorf("seeder %s failed: %w", seeder.Name(), err)
		}

		o.logger.Info(ctx, "seeder completed successfully", "seeder", seeder.Name())
	}

	o.logger.Info(ctx, "all seeders completed successfully")
	return nil
}
