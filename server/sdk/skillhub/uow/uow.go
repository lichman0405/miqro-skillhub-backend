// Package uow defines the unit-of-work / transaction boundary
// interface used by SDK services that need multi-repository atomicity.
//
// Source reference:
//
//	Spring @Transactional / PlatformTransactionManager in the Java
//	source.  Services that must run inside controlled transactions:
//	SkillPublishService, ReviewService, PromotionService, account
//	merge, token rotation, namespace ownership transfer, and storage
//	deletion compensation.
package uow

import "context"

// Transactor is the contract for beginning and committing/rolling
// back a database transaction that spans multiple repository calls.
//
// Storage object deletion must follow the source behavior:
//   - DB changes commit first.
//   - Object deletion happens after commit.
//   - Failed deletion records compensation.
type Transactor interface {
	// WithinTx executes fn inside a transaction.  If fn returns an
	// error the transaction is rolled back; otherwise it is committed.
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// NoopTransactor simply calls fn with the same context — no real
// transaction is started.  Suitable for tests that do not need a
// database.
type NoopTransactor struct{}

// WithinTx delegates to fn immediately.
func (NoopTransactor) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
