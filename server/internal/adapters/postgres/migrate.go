package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MigrationsDir is the filesystem path to the migrations directory.
// It is set at init time relative to the module root.
var MigrationsDir string

func init() {
	// Default: look for migrations relative to the working directory.
	// In tests and local runs, the working directory is typically server/.
	MigrationsDir = "migrations"
}

func resolveMigrationsDir() (string, error) {
	candidates := []string{
		MigrationsDir,
		filepath.Join("server", "migrations"),
	}

	if _, file, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(file), "..", "..", "..", "migrations"))
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if entries, err := os.ReadDir(candidate); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
					return candidate, nil
				}
			}
		}
	}

	return "", fmt.Errorf("migrations directory not found; tried %s", strings.Join(candidates, ", "))
}

// MigrateUp applies all SQL migration files in order from the filesystem.
// It tracks applied migrations in a schema_migrations table so that each
// migration is executed only once. Seed files that use DROP + INSERT should
// use ON CONFLICT DO NOTHING so re-running is safe.
func MigrateUp(ctx context.Context, pool *pgxpool.Pool) error {
	// Ensure tracking table exists.
	_, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("migrate: create tracking table: %w", err)
	}

	migrationsDir, err := resolveMigrationsDir()
	if err != nil {
		return fmt.Errorf("migrate: resolve migrations dir: %w", err)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("migrate: read dir %s: %w", migrationsDir, err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// Check if already applied.
		var applied bool
		err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, entry.Name()).Scan(&applied)
		if err != nil {
			return fmt.Errorf("migrate: check %s: %w", entry.Name(), err)
		}
		if applied {
			continue
		}

		filePath := filepath.Join(migrationsDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", entry.Name(), err)
		}

		if _, err := pool.Exec(ctx, string(data)); err != nil {
			return fmt.Errorf("migrate: exec %s: %w", entry.Name(), err)
		}

		// Record migration.
		if _, err := pool.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, entry.Name()); err != nil {
			return fmt.Errorf("migrate: record %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// Reset drops all tables and re-applies migrations from scratch.
// WARNING: Destroys all data. For development/test use only.
func Reset(ctx context.Context, pool *pgxpool.Pool) error {
	dropAll := `
	DO $$ DECLARE
	    r RECORD;
	BEGIN
	    FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
	        EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
	    END LOOP;
	END $$;
	`
	if _, err := pool.Exec(ctx, dropAll); err != nil {
		return fmt.Errorf("reset: drop tables: %w", err)
	}

	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS schema_migrations CASCADE`); err != nil {
		return fmt.Errorf("reset: drop schema_migrations: %w", err)
	}

	return MigrateUp(ctx, pool)
}
