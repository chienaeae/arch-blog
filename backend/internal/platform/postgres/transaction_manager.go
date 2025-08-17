package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TransactionManager provides transaction management capabilities
// This interface is defined here alongside its implementation for cohesion
type TransactionManager interface {
	BeginTx(ctx context.Context) (Transaction, error)
}

// Transaction represents a database transaction
type Transaction interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	Tx() pgx.Tx // Returns the underlying pgx.Tx for use with repository.WithTx
}

// PoolTransactionManager implements TransactionManager using a pgxpool.Pool
type PoolTransactionManager struct {
	pool *pgxpool.Pool
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(pool *pgxpool.Pool) TransactionManager {
	return &PoolTransactionManager{pool: pool}
}

// BeginTx starts a new database transaction
func (m *PoolTransactionManager) BeginTx(ctx context.Context) (Transaction, error) {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &PgxTransaction{tx: tx}, nil
}

// PgxTransaction wraps a pgx.Tx to implement the Transaction interface
type PgxTransaction struct {
	tx pgx.Tx
}

// Commit commits the transaction
func (t *PgxTransaction) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// Rollback rolls back the transaction
func (t *PgxTransaction) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

// Tx returns the underlying pgx.Tx for use with repository.WithTx
func (t *PgxTransaction) Tx() pgx.Tx {
	return t.tx
}