// Command skillhub-migrate runs database migrations.
//
// Usage:
//
//	skillhub-migrate          Apply pending migrations
//	skillhub-migrate reset    Drop all tables and re-apply migrations
//
// The DATABASE_URL environment variable must be set, or it defaults to
// the local Docker Compose connection string.
package main

import (
	"context"
	"fmt"
	"os"

	"miqro-skillhub/server/internal/adapters/postgres"
)

func main() {
	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		connString = "postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable"
	}

	ctx := context.Background()
	db, err := postgres.NewDB(ctx, connString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "skillhub-migrate: connect: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	doReset := len(os.Args) > 1 && os.Args[1] == "reset"

	if doReset {
		fmt.Fprintln(os.Stderr, "skillhub-migrate: resetting database and re-applying migrations...")
		if err := postgres.Reset(ctx, db.Pool); err != nil {
			fmt.Fprintf(os.Stderr, "skillhub-migrate: reset: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintln(os.Stderr, "skillhub-migrate: applying migrations...")
		if err := postgres.MigrateUp(ctx, db.Pool); err != nil {
			fmt.Fprintf(os.Stderr, "skillhub-migrate: migrate: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Fprintln(os.Stderr, "skillhub-migrate: done.")
}
