// Command skillhub-migrate runs database migrations.
//
// In Phase 01 this is a placeholder.  Migrations start in Phase 02
// with the PostgreSQL schema aligned to the source Flyway migrations
// under server/migrations/.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "skillhub-migrate: database migrations start in Phase 02.")
	os.Exit(0)
}
