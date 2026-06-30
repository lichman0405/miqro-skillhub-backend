package packagekit

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// SkillPackageValidator — validates skill packages before publish
// ---------------------------------------------------------------------------

// SkillPackageValidator validates skill package archives against structural,
// metadata, and size constraints. Mirrors source
// com.iflytek.skillhub.domain.skill.validation.SkillPackageValidator.
type SkillPackageValidator struct {
	parser         *SkillMetadataParser
	maxFileCount   int
	maxSingleSize  int64
	maxTotalSize   int64
	allowedExts    map[string]bool
}

// NewSkillPackageValidator creates a validator with default policy values.
func NewSkillPackageValidator(parser *SkillMetadataParser) *SkillPackageValidator {
	if parser == nil {
		parser = NewSkillMetadataParser()
	}
	return &SkillPackageValidator{
		parser:        parser,
		maxFileCount:  MaxFileCount,
		maxSingleSize: MaxSingleFileSize,
		maxTotalSize:  MaxTotalPackageSize,
		allowedExts:   AllowedExtensions,
	}
}

// Validate checks the provided package entries against all constraints.
// It collects errors and warnings separately; the result must have
// Passed() == true for the package to be publishable.
func (v *SkillPackageValidator) Validate(entries []PackageEntry) *ValidationResult {
	var errors []string
	var warnings []string
	seenPaths := make(map[string]bool)
	var skillMd *PackageEntry

	for i := range entries {
		entry := &entries[i]

		normalizedPath, errMsg := NormalizeEntryPath(entry.Path)
		if errMsg != "" {
			errors = append(errors, errMsg)
			continue
		}

		if seenPaths[normalizedPath] {
			errors = append(errors, "Duplicate package entry path: "+normalizedPath)
		}
		seenPaths[normalizedPath] = true

		if !HasAllowedExtension(normalizedPath) {
			warnings = append(warnings, "Disallowed file extension: "+normalizedPath)
		}

		if mismatch := ValidateContentMatchesExtension(normalizedPath, entry.Content); mismatch != "" {
			warnings = append(warnings, mismatch)
		}

		if normalizedPath == SkillMDPath && skillMd == nil {
			skillMd = entry
		}
	}

	// 1. SKILL.md must exist at root.
	if skillMd == nil {
		errors = append(errors, "Missing required file: SKILL.md at root")
		return &ValidationResult{Errors: errors, Warnings: warnings}
	}

	// 2. Validate frontmatter.
	content := string(skillMd.Content)
	if _, err := v.parser.Parse(content); err != nil {
		errors = append(errors, "Invalid SKILL.md frontmatter: "+formatMetadataError(err))
	}

	// 3. Check file count.
	if len(entries) > v.maxFileCount {
		errors = append(errors, fmt.Sprintf("Too many files: %d (max: %d)", len(entries), v.maxFileCount))
	}

	// 4. Check single file sizes.
	for i := range entries {
		entry := &entries[i]
		if entry.Size > v.maxSingleSize {
			errors = append(errors, fmt.Sprintf(
				"File too large: %s (%d bytes, max: %d)", entry.Path, entry.Size, v.maxSingleSize))
		}
	}

	// 5. Check total package size.
	var totalSize int64
	for _, e := range entries {
		totalSize += e.Size
	}
	if totalSize > v.maxTotalSize {
		errors = append(errors, fmt.Sprintf(
			"Package too large: %d bytes (max: %d)", totalSize, v.maxTotalSize))
	}

	return &ValidationResult{Errors: errors, Warnings: warnings}
}

// formatMetadataError converts parser errors to the human-readable format
// used by the source SkillPackageValidator.formatMetadataError.
func formatMetadataError(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "requiredField.missing"):
		parts := strings.SplitN(msg, " ", 2)
		field := ""
		if len(parts) > 1 {
			field = parts[1]
		}
		return fmt.Sprintf("missing required field \"%s\"", field)
	case strings.Contains(msg, "missingStart"):
		return "missing opening --- marker"
	case strings.Contains(msg, "missingEnd"):
		return "missing closing --- marker"
	case strings.Contains(msg, "missingContent"):
		return "frontmatter is empty"
	case strings.Contains(msg, "notMap"):
		return "frontmatter must be a YAML object"
	default:
		if strings.Contains(msg, "yaml.invalid") {
			return "invalid YAML syntax. If a value contains a colon, wrap it in quotes."
		}
		return msg
	}
}
