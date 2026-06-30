package packagekit

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var (
	ErrMissingStart   = errors.New("error.skill.metadata.frontmatter.missingStart")
	ErrMissingEnd     = errors.New("error.skill.metadata.frontmatter.missingEnd")
	ErrMissingContent = errors.New("error.skill.metadata.frontmatter.missingContent")
	ErrEmptyContent   = errors.New("error.skill.metadata.content.empty")
	ErrNotMap         = errors.New("error.skill.metadata.yaml.notMap")
)

// ---------------------------------------------------------------------------
// SkillMetadataParser
// ---------------------------------------------------------------------------

const frontmatterDelim = "---"

// SkillMetadataParser parses YAML frontmatter and body from SKILL.md content.
// Mirrors source com.iflytek.skillhub.domain.skill.metadata.SkillMetadataParser.
type SkillMetadataParser struct {
	yamlDecoder func([]byte, any) error
}

// NewSkillMetadataParser creates a parser with the default YAML decoder (gopkg.in/yaml.v3).
func NewSkillMetadataParser() *SkillMetadataParser {
	return &SkillMetadataParser{
		yamlDecoder: yaml.Unmarshal,
	}
}

// Parse extracts the YAML frontmatter and body from SKILL.md content.
// Required fields: name, description. Optional: version (or metadata.version).
func (p *SkillMetadataParser) Parse(content string) (*SkillMetadata, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("%w", ErrEmptyContent)
	}

	trimmed := strings.TrimSpace(content)

	// Frontmatter must start with "---".
	if !strings.HasPrefix(trimmed, frontmatterDelim) {
		return nil, fmt.Errorf("%w", ErrMissingStart)
	}

	// Find end of first delimiter line.
	firstEnd := strings.Index(trimmed, "\n")
	if firstEnd == -1 {
		return nil, fmt.Errorf("%w", ErrMissingContent)
	}

	// Find second delimiter.
	secondStart := strings.Index(trimmed[firstEnd+1:], frontmatterDelim)
	if secondStart == -1 {
		return nil, fmt.Errorf("%w", ErrMissingEnd)
	}
	secondStart += firstEnd + 1

	yamlContent := strings.TrimSpace(trimmed[firstEnd+1 : secondStart])
	body := strings.TrimSpace(trimmed[secondStart+len(frontmatterDelim):])

	// Parse YAML frontmatter.
	frontmatter, err := p.parseFrontmatter(yamlContent)
	if err != nil {
		// Try loose parsing as fallback (mirrors source parseLooseFrontmatter).
		loose := parseLooseFrontmatter(yamlContent)
		if len(loose) > 0 {
			frontmatter = loose
		} else {
			return nil, err
		}
	}

	// Extract required fields.
	name, err := extractRequiredField(frontmatter, "name")
	if err != nil {
		return nil, err
	}
	description, err := extractRequiredField(frontmatter, "description")
	if err != nil {
		return nil, err
	}

	// Extract version (optional; may be nested in metadata.version).
	version := extractOptionalField(frontmatter, "version")
	if version == "" {
		version = extractNestedOptionalField(frontmatter, "metadata", "version")
	}

	return &SkillMetadata{
		Name:        name,
		Description: description,
		Version:     version,
		Body:        body,
		Frontmatter: frontmatter,
	}, nil
}

func (p *SkillMetadataParser) parseFrontmatter(yamlContent string) (map[string]any, error) {
	var result map[string]any
	if err := p.yamlDecoder([]byte(yamlContent), &result); err != nil {
		return nil, fmt.Errorf("error.skill.metadata.yaml.invalid %s", err.Error())
	}
	if result == nil {
		return nil, fmt.Errorf("%w", ErrNotMap)
	}
	return result, nil
}

// parseLooseFrontmatter does key:value line-by-line parsing as a fallback.
// Mirrors source SkillMetadataParser.parseLooseFrontmatter.
func parseLooseFrontmatter(yamlContent string) map[string]any {
	values := make(map[string]any)
	for _, rawLine := range strings.Split(yamlContent, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		if key == "" {
			continue
		}

		values[key] = stripWrappingQuotes(value)
	}
	return values
}

func stripWrappingQuotes(value string) string {
	if len(value) >= 2 {
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func extractRequiredField(frontmatter map[string]any, fieldName string) (string, error) {
	val, ok := frontmatter[fieldName]
	if !ok {
		return "", fmt.Errorf("error.skill.metadata.requiredField.missing %s", fieldName)
	}
	return fmt.Sprintf("%v", val), nil
}

func extractOptionalField(frontmatter map[string]any, fieldName string) string {
	val, ok := frontmatter[fieldName]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

func extractNestedOptionalField(frontmatter map[string]any, objectFieldName, fieldName string) string {
	nested, ok := frontmatter[objectFieldName]
	if !ok {
		return ""
	}
	nestedMap, ok := nested.(map[string]any)
	if !ok {
		return ""
	}
	return extractOptionalField(nestedMap, fieldName)
}
