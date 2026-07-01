package tooling

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"miqro-skillhub/server/sdk/skillhub/packagekit"
)

// BuildManifest creates a deterministic package manifest from a set of
// package entries.  Entries are sorted by path, and each entry includes
// its SHA-256 hash.
func BuildManifest(entries []packagekit.PackageEntry) PackageManifest {
	var manifestEntries []ManifestEntry
	var totalSize int64

	// Sort entries by path for determinism.
	sorted := make([]packagekit.PackageEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Path < sorted[j].Path
	})

	for _, entry := range sorted {
		hash := sha256.Sum256(entry.Content)
		sha256Hex := hex.EncodeToString(hash[:])
		manifestEntries = append(manifestEntries, ManifestEntry{
			Path:        entry.Path,
			Size:        entry.Size,
			ContentType: entry.ContentType,
			SHA256:      sha256Hex,
		})
		totalSize += entry.Size
	}

	// Compute the manifest hash: SHA-256 over sorted "path:sha256\n" lines.
	manifestHash := ComputeManifestHash(manifestEntries)

	return PackageManifest{
		Entries:   manifestEntries,
		Hash:      manifestHash,
		TotalSize: totalSize,
		FileCount: len(manifestEntries),
	}
}

// ComputeManifestHash computes the deterministic package hash from
// a sorted list of manifest entries.  The hash is the SHA-256 digest
// of the concatenation of "path:sha256\n" for each entry, prefixed
// with "sha256:".
func ComputeManifestHash(entries []ManifestEntry) string {
	h := sha256.New()
	for _, e := range entries {
		fmt.Fprintf(h, "%s:%s\n", e.Path, e.SHA256)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}
