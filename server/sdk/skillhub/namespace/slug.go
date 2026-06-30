package namespace

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// ---------------------------------------------------------------------------
// Slug validation constraints
// ---------------------------------------------------------------------------

const (
	SlugMinLength = 2
	SlugMaxLength = 64
)

// Slug validation error codes (machine-readable constants).
const (
	ErrCodeSlugTooShort      = "namespace.slug.too_short"
	ErrCodeSlugTooLong       = "namespace.slug.too_long"
	ErrCodeSlugReserved      = "namespace.slug.reserved"
	ErrCodeSlugInvalidFormat = "namespace.slug.invalid_format"
	ErrCodeSlugUppercase     = "namespace.slug.uppercase"
	ErrCodeSlugDoubleHyphen  = "namespace.slug.double_hyphen"
)

// Sentinel errors for slug validation.
var (
	ErrInvalidSlug = errors.New("invalid slug")
)

// reservedSlugs is the set of slugs that cannot be used for user-created namespaces.
var reservedSlugs = map[string]bool{
	"admin":     true,
	"api":       true,
	"dashboard": true,
	"search":    true,
	"auth":      true,
	"me":        true,
	"global":    true,
	"system":    true,
	"static":    true,
	"assets":    true,
	"health":    true,
}

// slugPattern matches a slug that starts with a Unicode letter, number, or symbol,
// may contain those plus hyphens in the middle, and ends with a letter, number, or symbol.
var slugPattern = regexp.MustCompile(`^[\p{L}\p{N}\p{S}][\p{L}\p{N}\p{S}-]*[\p{L}\p{N}\p{S}]$`)

// nonSlugPattern matches characters that are not allowed in a slug.
var nonSlugPattern = regexp.MustCompile(`[^\p{L}\p{N}-]`)

// multiHyphenPattern matches consecutive hyphens.
var multiHyphenPattern = regexp.MustCompile(`-{2,}`)

// ---------------------------------------------------------------------------
// SlugValidator — structured validator returning error codes
// ---------------------------------------------------------------------------

// SlugValidator validates namespace slugs against business constraints.
type SlugValidator struct{}

// Validate checks whether a slug meets all constraints.
// Returns an error code string if invalid, or an empty string if valid.
func (v SlugValidator) Validate(slug string) string {
	if len(slug) < SlugMinLength {
		return ErrCodeSlugTooShort
	}
	if len(slug) > SlugMaxLength {
		return ErrCodeSlugTooLong
	}
	if reservedSlugs[slug] {
		return ErrCodeSlugReserved
	}
	// Reject uppercase characters.
	for _, r := range slug {
		if unicode.IsUpper(r) {
			return ErrCodeSlugUppercase
		}
	}
	// Reject double hyphens.
	if strings.Contains(slug, "--") {
		return ErrCodeSlugDoubleHyphen
	}
	// Validate pattern: start with letter/number/symbol, middle may include hyphens,
	// end with letter/number/symbol.
	if !slugPattern.MatchString(slug) {
		return ErrCodeSlugInvalidFormat
	}
	return ""
}

// ---------------------------------------------------------------------------
// Package-level convenience functions (used by tests and callers)
// ---------------------------------------------------------------------------

// Validate checks whether a slug is valid according to namespace rules.
// It returns a descriptive error or nil.
func Validate(slug string) error {
	if len(slug) < 2 {
		return fmt.Errorf("%w: slug must be at least 2 characters", ErrInvalidSlug)
	}
	if len(slug) > 64 {
		return fmt.Errorf("%w: slug must be at most 64 characters", ErrInvalidSlug)
	}
	if reservedSlugs[slug] {
		return fmt.Errorf("%w: %q is reserved", ErrInvalidSlug, slug)
	}
	// Check uppercase before pattern match (slugPattern's \p{L} includes uppercase).
	for _, ch := range slug {
		if ch >= 'A' && ch <= 'Z' {
			return fmt.Errorf("%w: uppercase letters are not allowed", ErrInvalidSlug)
		}
	}
	// Check underscore.
	if strings.Contains(slug, "_") {
		return fmt.Errorf("%w: underscores are not allowed", ErrInvalidSlug)
	}
	// Check leading/trailing/consecutive hyphens before pattern match
	// (slugPattern's character class for middle includes "-").
	if strings.HasPrefix(slug, "-") {
		return fmt.Errorf("%w: leading hyphen is not allowed", ErrInvalidSlug)
	}
	if strings.HasSuffix(slug, "-") {
		return fmt.Errorf("%w: trailing hyphen is not allowed", ErrInvalidSlug)
	}
	if strings.Contains(slug, "--") {
		return fmt.Errorf("%w: double hyphen is not allowed", ErrInvalidSlug)
	}
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("%w: invalid characters in %q", ErrInvalidSlug, slug)
	}
	return nil
}

// Slugify converts an arbitrary string into a valid slug by trimming whitespace,
// lowercasing, replacing non-alphanumeric characters with hyphens, collapsing
// consecutive hyphens, and trimming leading/trailing hyphens.
func Slugify(input string) (string, error) {
	// Trim whitespace and lowercase.
	s := strings.TrimSpace(strings.ToLower(input))

	// Replace characters not allowed in a slug with hyphens.
	s = nonSlugPattern.ReplaceAllString(s, "-")

	// Collapse consecutive hyphens.
	s = multiHyphenPattern.ReplaceAllString(s, "-")

	// Trim leading and trailing hyphens.
	s = strings.Trim(s, "-")

	if len(s) < 2 {
		return "", fmt.Errorf("%w: slugify produced a slug shorter than 2 characters", ErrInvalidSlug)
	}
	if len(s) > 64 {
		s = s[:64]
	}
	// Trim hyphens that may have ended up at boundaries after truncation.
	s = strings.Trim(s, "-")

	return s, nil
}
