package packagekit

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// Package policy constants — mirror source SkillPackagePolicy
// ---------------------------------------------------------------------------

const (
	// MaxFileCount is the maximum number of files allowed in a package.
	MaxFileCount = 500
	// MaxSingleFileSize is the maximum size in bytes for a single file (10 MB).
	MaxSingleFileSize = 10 * 1024 * 1024
	// MaxTotalPackageSize is the maximum total size in bytes for all files (100 MB).
	MaxTotalPackageSize = 100 * 1024 * 1024
	// SkillMDPath is the required root manifest filename.
	SkillMDPath = "SKILL.md"
)

// AllowedExtensions is the set of permitted file extensions.
// Mirrors source SkillPackagePolicy.ALLOWED_EXTENSIONS exactly.
var AllowedExtensions = map[string]bool{
	// Documentation
	".md": true, ".txt": true, ".json": true, ".yaml": true, ".yml": true,
	".html": true, ".css": true, ".csv": true, ".pdf": true,
	// Configuration and schemas
	".toml": true, ".xml": true, ".xsd": true, ".xsl": true, ".dtd": true,
	".ini": true, ".cfg": true, ".env": true,
	// Scripts and source code
	".js": true, ".cjs": true, ".mjs": true, ".ts": true, ".py": true,
	".sh": true, ".rb": true, ".go": true, ".rs": true, ".java": true,
	".kt": true, ".lua": true, ".sql": true, ".r": true,
	".bat": true, ".ps1": true, ".zsh": true, ".bash": true,
	// Images
	".png": true, ".jpg": true, ".jpeg": true, ".svg": true,
	".gif": true, ".webp": true, ".ico": true,
	// Office documents
	".doc": true, ".xls": true, ".ppt": true,
	".docx": true, ".xlsx": true, ".pptx": true,
}

// textExtensions is the set of extensions that should be valid UTF-8 text.
var textExtensions = map[string]bool{
	".md": true, ".txt": true,
	".json": true, ".yaml": true, ".yml": true,
	".js": true, ".cjs": true, ".mjs": true,
	".ts": true, ".py": true, ".sh": true,
	".html": true, ".css": true, ".csv": true,
	".toml": true, ".xml": true, ".xsd": true,
	".xsl": true, ".dtd": true, ".ini": true,
	".cfg": true, ".env": true,
	".rb": true, ".go": true, ".rs": true,
	".java": true, ".kt": true, ".lua": true,
	".sql": true, ".r": true,
	".bat": true, ".ps1": true,
	".zsh": true, ".bash": true,
}

// ---------------------------------------------------------------------------
// Path normalization — mirror source SkillPackagePolicy.normalizeEntryPath
// ---------------------------------------------------------------------------

// NormalizeEntryPath validates and normalizes a package entry path.
//   - Rejects absolute paths, drive/scheme prefixes, ".." escape.
//   - Normalizes backslashes to forward slashes.
//   - Canonicalizes SKILL.md filename (case-insensitive match).
//   - Returns an error message string if invalid.
func NormalizeEntryPath(rawPath string) (string, string) {
	if rawPath == "" {
		return "", "Package entry path is missing"
	}

	sanitized := strings.ReplaceAll(rawPath, "\\", "/")
	sanitized = strings.TrimSpace(sanitized)
	if sanitized == "" {
		return "", "Package entry path is empty"
	}
	if strings.HasPrefix(sanitized, "/") {
		return "", "Package entry path must be relative: " + rawPath
	}
	if strings.Contains(sanitized, ":") {
		return "", "Package entry path contains an invalid drive or scheme prefix: " + rawPath
	}

	// Clean the path (resolves ".", "..", "//").
	cleaned := filepath.ToSlash(filepath.Clean(sanitized))
	if strings.HasPrefix(cleaned, "/") {
		return "", "Package entry path is invalid: " + rawPath
	}
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", "Package entry path escapes package root: " + rawPath
	}
	if sanitized != cleaned {
		return "", "Package entry path must be normalized: " + rawPath
	}

	return canonicalizeSkillMdPath(cleaned), ""
}

// canonicalizeSkillMdPath preserves the exact "SKILL.md" casing at root or in
// subdirectories. Mirrors source SkillPackagePolicy.canonicalizeSkillMdPath.
func canonicalizeSkillMdPath(normalizedPath string) string {
	idx := strings.LastIndex(normalizedPath, "/")
	fileName := normalizedPath
	if idx >= 0 {
		fileName = normalizedPath[idx+1:]
	}
	if !strings.EqualFold(fileName, SkillMDPath) {
		return normalizedPath
	}
	if idx < 0 {
		return SkillMDPath
	}
	return normalizedPath[:idx+1] + SkillMDPath
}

// HasAllowedExtension reports whether the path ends with a permitted extension.
func HasAllowedExtension(path string) bool {
	for ext := range AllowedExtensions {
		if strings.HasSuffix(strings.ToLower(path), ext) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Content signature checks — mirror source SkillPackagePolicy
// ---------------------------------------------------------------------------

// ValidateContentMatchesExtension performs lightweight file signature checks.
// Returns an empty string on success, or a warning message on mismatch.
func ValidateContentMatchesExtension(path string, content []byte) string {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".png"):
		if !hasPrefix(content, 0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a) {
			return fmt.Sprintf("File content does not match extension: %s", path)
		}
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		if !hasPrefix(content, 0xff, 0xd8, 0xff) {
			return fmt.Sprintf("File content does not match extension: %s", path)
		}
	case strings.HasSuffix(lower, ".gif"):
		if !hasPrefix(content, 'G', 'I', 'F', '8') {
			return fmt.Sprintf("File content does not match extension: %s", path)
		}
	case strings.HasSuffix(lower, ".webp"):
		if !(len(content) >= 12 &&
			hasPrefix(content, 'R', 'I', 'F', 'F') &&
			content[8] == 'W' && content[9] == 'E' && content[10] == 'B' && content[11] == 'P') {
			return fmt.Sprintf("File content does not match extension: %s", path)
		}
	case strings.HasSuffix(lower, ".ico"):
		if !hasPrefix(content, 0x00, 0x00, 0x01, 0x00) {
			return fmt.Sprintf("File content does not match extension: %s", path)
		}
	case strings.HasSuffix(lower, ".pdf"):
		if !hasPrefix(content, '%', 'P', 'D', 'F') {
			return fmt.Sprintf("File content does not match extension: %s", path)
		}
	case strings.HasSuffix(lower, ".svg"):
		if !isUtf8Text(content) {
			return fmt.Sprintf("File content does not match extension: %s", path)
		}
		text := strings.TrimSpace(string(content))
		if !strings.Contains(strings.ToLower(text), "<svg") {
			return fmt.Sprintf("File content does not match extension: %s", path)
		}
	default:
		if isTextExtension(lower) {
			if !isUtf8Text(content) {
				return fmt.Sprintf("File content does not match extension: %s", path)
			}
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func hasPrefix(content []byte, prefix ...int) bool {
	if len(content) < len(prefix) {
		return false
	}
	for i, p := range prefix {
		if int(content[i])&0xff != p&0xff {
			return false
		}
	}
	return true
}

func isTextExtension(lowerPath string) bool {
	for ext := range textExtensions {
		if strings.HasSuffix(lowerPath, ext) {
			return true
		}
	}
	return false
}

func isUtf8Text(content []byte) bool {
	// Reject null bytes (source behavior).
	for _, b := range content {
		if b == 0 {
			return false
		}
	}
	return utf8.Valid(content)
}
