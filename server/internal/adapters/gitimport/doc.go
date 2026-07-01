// Package gitimport provides adapter support for importing skill packages
// from external Git repositories.  It is intentionally a thin adapter layer —
// Git is not the core model for miqro-skillhub.  The core model is skill-native
// package hosting.
//
// Full Git import implementation belongs to a future phase.  This package
// exists to establish the adapter path so that Git-backed publishing never
// bleeds into core SDK packages.
package gitimport
