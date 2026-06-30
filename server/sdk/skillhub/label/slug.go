package label

import (
	"fmt"
	"regexp"
	"strings"
)

// LabelType constants.
const (
	TypeRecommended = "RECOMMENDED"
	TypePrivileged  = "PRIVILEGED"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidateSlug validates a label slug.
func ValidateSlug(slug string) error {
	normalized := strings.ToLower(strings.TrimSpace(slug))
	if normalized == "" {
		return fmt.Errorf("error.label.slug.empty")
	}
	if len(normalized) > 64 {
		return fmt.Errorf("error.label.slug.too_long")
	}
	if !slugPattern.MatchString(normalized) {
		return fmt.Errorf("error.label.slug.invalid %s", normalized)
	}
	return nil
}
