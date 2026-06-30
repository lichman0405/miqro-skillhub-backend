package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

// MigrateUp applies all SQL migration files in order from the filesystem.
func MigrateUp(ctx context.Context, pool *pgxpool.Pool) error {
	entries, err := os.ReadDir(MigrationsDir)
	if err != nil {
		return fmt.Errorf("migrate: read dir %s: %w", MigrationsDir, err)
	}

	// Sort by filename to ensure 001 < 002 < 003 < 004 < 005
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		filePath := filepath.Join(MigrationsDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", entry.Name(), err)
		}

		if _, err := pool.Exec(ctx, string(data)); err != nil {
			return fmt.Errorf("migrate: exec %s: %w", entry.Name(), err)
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

	return MigrateUp(ctx, pool)
}
