package tooling

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"miqro-skillhub/server/sdk/skillhub/skill"
)

// ComputeVersionFingerprint computes a SHA-256 fingerprint for a sorted
// list of skill files.  The computation mirrors:
//
//	source SkillQueryService.computeFingerprint
//
// It concatenates "path:sha256\n" for each file (sorted by path), hashes
// that with SHA-256, and returns the hex digest prefixed with "sha256:".
func ComputeVersionFingerprint(files []skill.SkillFile) string {
	// Sort by file path.
	sorted := make([]skill.SkillFile, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].FilePath < sorted[j].FilePath
	})

	h := sha256.New()
	for _, f := range sorted {
		fmt.Fprintf(h, "%s:%s\n", f.FilePath, f.SHA256)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}
