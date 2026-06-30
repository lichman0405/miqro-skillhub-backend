package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Transactor implements the uow.Transactor interface using PostgreSQL transactions.
type Transactor struct {
	Pool *pgxpool.Pool
}

// NewTransactor creates a new Transactor.
func NewTransactor(pool *pgxpool.Pool) *Transactor {
	return &Transactor{Pool: pool}
}

// WithinTx executes fn within a PostgreSQL transaction. If fn returns an error,
// the transaction is rolled back. Otherwise it is committed.
func (t *Transactor) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := t.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("transactor: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := fn(ctx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("transactor: commit: %w", err)
	}

	return nil
}

// txKey is the context key for the transaction.
type txKey struct{}

// TxFromContext extracts the pgx.Tx from context, or returns nil.
func TxFromContext(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(txKey{}).(pgx.Tx)
	return tx
}

// WithTx stores the transaction in context.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}
