// Package postgres provides PostgreSQL adapter implementations for SDK repository interfaces.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgxpool.Pool and provides helpers for common operations.
type DB struct {
	Pool *pgxpool.Pool
}

// NewDB creates a new DB from a connection string.
func NewDB(ctx context.Context, connString string) (*DB, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("postgres: connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close closes the database connection pool.
func (db *DB) Close() {
	db.Pool.Close()
}

// exec runs an Exec against either the context transaction (if present) or the pool.
func (db *DB) exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if tx := TxFromContext(ctx); tx != nil {
		return tx.Exec(ctx, sql, args...)
	}
	return db.Pool.Exec(ctx, sql, args...)
}

// query runs a Query against either the context transaction (if present) or the pool.
func (db *DB) query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if tx := TxFromContext(ctx); tx != nil {
		return tx.Query(ctx, sql, args...)
	}
	return db.Pool.Query(ctx, sql, args...)
}

// queryRow runs a QueryRow against either the context transaction (if present) or the pool.
func (db *DB) queryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if tx := TxFromContext(ctx); tx != nil {
		return tx.QueryRow(ctx, sql, args...)
	}
	return db.Pool.QueryRow(ctx, sql, args...)
}
