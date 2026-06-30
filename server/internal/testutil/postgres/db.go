// Package postgres provides test helpers for PostgreSQL integration tests.
package postgres

import (
	"context"
	"os"
	"testing"

	"miqro-skillhub/server/internal/adapters/postgres"
)

// TestDB returns a DB connected to the test database. If SKILLHUB_TEST_DATABASE_URL
// is set, it uses that. Otherwise it uses a default local connection.
// If no PostgreSQL is available, the test is skipped.
func TestDB(t testing.TB) *postgres.DB {
	t.Helper()

	connString := os.Getenv("SKILLHUB_TEST_DATABASE_URL")
	if connString == "" {
		connString = "postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable"
	}

	ctx := context.Background()
	db, err := postgres.NewDB(ctx, connString)
	if err != nil {
		t.Skipf("PostgreSQL not available for integration test: %v", err)
		return nil
	}

	// Reset the schema to a known state.
	if err := postgres.Reset(ctx, db.Pool); err != nil {
		db.Close()
		t.Fatalf("Failed to reset database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}
