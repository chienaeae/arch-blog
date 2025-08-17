package postgres

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier is a common interface for both pgxpool.Pool and pgx.Tx
// This allows us to support both regular queries and transactions
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

// BaseRepository contains the common database components that all repositories need
type BaseRepository struct {
	DB Querier                  // Database connection (pool or transaction)
	SB sq.StatementBuilderType  // SQL builder with PostgreSQL placeholders
}

// NewBaseRepository creates a new base repository with a database pool
func NewBaseRepository(db *pgxpool.Pool) BaseRepository {
	return BaseRepository{
		DB: db,
		SB: sq.StatementBuilder.PlaceholderFormat(sq.Dollar), // PostgreSQL $1, $2 placeholders
	}
}

// WithTx creates a new BaseRepository that uses the provided transaction
func (b BaseRepository) WithTx(tx pgx.Tx) BaseRepository {
	return BaseRepository{
		DB: tx,
		SB: b.SB, // Keep the same statement builder configuration
	}
}