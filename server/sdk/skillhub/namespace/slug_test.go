package namespace_test

import (
	"testing"

	"miqro-skillhub/server/sdk/skillhub/namespace"
)

func TestValidateValidSlugs(t *testing.T) {
	valid := []string{"my-namespace", "ab", "test123", "my-team-2024"}
	for _, s := range valid {
		if err := namespace.Validate(s); err != nil {
			t.Errorf("expected valid, got error for %q: %v", s, err)
		}
	}
}

func TestValidateTooShort(t *testing.T) {
	if err := namespace.Validate("a"); err == nil {
		t.Error("expected error for too short")
	}
}

func TestValidateTooLong(t *testing.T) {
	long := ""
	for i := 0; i < 65; i++ {
		long += "x"
	}
	if err := namespace.Validate(long); err == nil {
		t.Error("expected error for too long")
	}
}

func TestValidateUppercase(t *testing.T) {
	if err := namespace.Validate("MyNamespace"); err == nil {
		t.Error("expected error for uppercase")
	}
}

func TestValidateLeadingHyphen(t *testing.T) {
	if err := namespace.Validate("-namespace"); err == nil {
		t.Error("expected error for leading hyphen")
	}
}

func TestValidateTrailingHyphen(t *testing.T) {
	if err := namespace.Validate("namespace-"); err == nil {
		t.Error("expected error for trailing hyphen")
	}
}

func TestValidateDoubleHyphen(t *testing.T) {
	if err := namespace.Validate("my--namespace"); err == nil {
		t.Error("expected error for double hyphen")
	}
}

func TestValidateReservedSlug(t *testing.T) {
	for _, s := range []string{"admin", "api", "global", "system"} {
		if err := namespace.Validate(s); err == nil {
			t.Errorf("expected error for reserved slug %q", s)
		}
	}
}

func TestValidateUnderscore(t *testing.T) {
	if err := namespace.Validate("my_namespace"); err == nil {
		t.Error("expected error for underscore")
	}
}

func TestValidateUnicode(t *testing.T) {
	valid := []string{"技能包", "my-技能-v2", "スキル", "테스트-skill"}
	for _, s := range valid {
		if err := namespace.Validate(s); err != nil {
			t.Errorf("expected valid unicode, got error: %v", err)
		}
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"My Skill Package!", "my-skill-package"},
		{"  Hello World  ", "hello-world"},
		{"UPPERCASE", "uppercase"},
		{"a--b--c", "a-b-c"},
		{"-leading", "leading"},
		{"trailing-", "trailing"},
	}
	for _, tt := range tests {
		result, err := namespace.Slugify(tt.input)
		if err != nil {
			t.Errorf("Slugify(%q) error: %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("Slugify(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
