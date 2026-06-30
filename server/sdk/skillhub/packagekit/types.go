package packagekit

// ---------------------------------------------------------------------------
// Package entry and validation result types
// ---------------------------------------------------------------------------

// PackageEntry represents a single file within a skill package archive.
// It mirrors source com.iflytek.skillhub.domain.skill.validation.PackageEntry.
type PackageEntry struct {
	Path        string // normalized relative path (always forward-slash)
	Content     []byte
	Size        int64
	ContentType string
}

// ValidationResult is the outcome of package validation.
// It mirrors source com.iflytek.skillhub.domain.skill.validation.ValidationResult.
type ValidationResult struct {
	Errors   []string
	Warnings []string
}

// Passed returns true when there are no errors.
func (r ValidationResult) Passed() bool {
	return len(r.Errors) == 0
}

// Valid returns true when both error and warning lists are empty (used by dry-run).
func (r ValidationResult) Valid() bool {
	return len(r.Errors) == 0 && len(r.Warnings) == 0
}

// ---------------------------------------------------------------------------
// SKILL.md metadata
// ---------------------------------------------------------------------------

// SkillMetadata holds the parsed frontmatter and body from SKILL.md.
// It mirrors source com.iflytek.skillhub.domain.skill.metadata.SkillMetadata.
type SkillMetadata struct {
	Name        string
	Description string
	Version     string // may be empty (auto-generated on publish)
	Body        string
	Frontmatter map[string]any
}
