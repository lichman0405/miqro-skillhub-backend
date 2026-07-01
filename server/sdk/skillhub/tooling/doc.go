// Package tooling provides SDK-level support for the miqro skill-native
// toolchain.  It wraps existing skill/publish/download/query services and
// adds deterministic hashing, version fingerprinting, package diff support,
// workspace metadata, install-target resolution, and placeholder protocols
// for evaluation and proposals.
//
// The package is SDK-first: core types and algorithms have no HTTP
// dependency.  HTTP adapters live under internal/http/toolapi.
//
// Git import/export is deliberately absent from this package — it is an
// adapter concern (internal/adapters/gitimport), not part of the core model.
package tooling
