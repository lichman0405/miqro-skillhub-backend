// Command skillhub-worker runs background job processing.
//
// Workers start in later phases:
//   - Phase 03: auth session cleanup
//   - Phase 05: storage deletion compensation
//   - Phase 06: notification dispatch, audit log writes
//   - Phase 07: search index rebuild, security scan callbacks
//
// In Phase 01 this is a placeholder that exits cleanly.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "skillhub-worker: background workers start in later phases.")
	os.Exit(0)
}
