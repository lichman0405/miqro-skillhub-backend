// Package toolapi implements the tool-facing HTTP API surface.
//
// Routes under /api/tool/v1/ are designed for the miqro CLI and other
// automated tool consumers.  They are separate from the frontend page
// read-models (/api/v1/frontend/*) and focus on deterministic metadata,
// hashes, fingerprints, diffs, and install targets.
//
// This package is an adapter — all core behavior lives in the tooling SDK
// package (server/sdk/skillhub/tooling).
package toolapi

import (
	"miqro-skillhub/server/sdk/skillhub/tooling"
)

// Handler exposes /api/tool/v1/* tool-facing routes.
type Handler struct {
	Tooling *tooling.Service
}
