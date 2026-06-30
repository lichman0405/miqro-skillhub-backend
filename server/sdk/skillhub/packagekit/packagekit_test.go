package packagekit_test

import (
	"strings"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/packagekit"
)

// ============================================================================
// Path normalization tests
// ============================================================================

func TestNormalizeEntryPath_SimplePath(t *testing.T) {
	path, errMsg := packagekit.NormalizeEntryPath("README.md")
	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	if path != "README.md" {
		t.Errorf("expected 'README.md', got '%s'", path)
	}
}

func TestNormalizeEntryPath_NestedPath(t *testing.T) {
	path, errMsg := packagekit.NormalizeEntryPath("src/main.go")
	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	if path != "src/main.go" {
		t.Errorf("expected 'src/main.go', got '%s'", path)
	}
}

func TestNormalizeEntryPath_BackslashToForward(t *testing.T) {
	path, errMsg := packagekit.NormalizeEntryPath("src\\main.go")
	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	if path != "src/main.go" {
		t.Errorf("expected 'src/main.go', got '%s'", path)
	}
}

func TestNormalizeEntryPath_CanonicalizesSKILLMD(t *testing.T) {
	path, errMsg := packagekit.NormalizeEntryPath("skill.md")
	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	if path != "SKILL.md" {
		t.Errorf("expected 'SKILL.md', got '%s'", path)
	}

	path, errMsg = packagekit.NormalizeEntryPath("SKILL.MD")
	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	if path != "SKILL.md" {
		t.Errorf("expected 'SKILL.md', got '%s'", path)
	}
}

func TestNormalizeEntryPath_AbsolutePathRejected(t *testing.T) {
	_, errMsg := packagekit.NormalizeEntryPath("/etc/passwd")
	if errMsg == "" {
		t.Fatal("expected error for absolute path")
	}
	if !strings.Contains(errMsg, "must be relative") {
		t.Errorf("expected 'must be relative', got '%s'", errMsg)
	}
}

func TestNormalizeEntryPath_PathEscapeRejected(t *testing.T) {
	_, errMsg := packagekit.NormalizeEntryPath("../escape.sh")
	if errMsg == "" {
		t.Fatal("expected error for escape path")
	}
	if !strings.Contains(errMsg, "escapes package root") {
		t.Errorf("expected 'escapes package root', got '%s'", errMsg)
	}
}

func TestNormalizeEntryPath_DrivePrefixRejected(t *testing.T) {
	_, errMsg := packagekit.NormalizeEntryPath("C:\\file.txt")
	if errMsg == "" {
		t.Fatal("expected error for drive prefix")
	}
}

func TestNormalizeEntryPath_DoubleDotEscape(t *testing.T) {
	_, errMsg := packagekit.NormalizeEntryPath("a/../../secret.txt")
	if errMsg == "" {
		t.Fatal("expected error for double-dot escape")
	}
}

func TestNormalizeEntryPath_UnnormalizedPath(t *testing.T) {
	_, errMsg := packagekit.NormalizeEntryPath("a//b/c/./d")
	if errMsg == "" {
		t.Fatal("expected error for unnormalized path")
	}
	if !strings.Contains(errMsg, "must be normalized") {
		t.Errorf("expected 'must be normalized', got '%s'", errMsg)
	}
}

func TestNormalizeEntryPath_EmptyPath(t *testing.T) {
	_, errMsg := packagekit.NormalizeEntryPath("")
	if errMsg == "" {
		t.Fatal("expected error for empty path")
	}
}

// ============================================================================
// Extension allowlist tests
// ============================================================================

func TestHasAllowedExtension_KnownExtensions(t *testing.T) {
	for _, ext := range []string{".md", ".py", ".go", ".json", ".png", ".txt", ".yaml", ".pdf", ".html", ".ts"} {
		if !packagekit.HasAllowedExtension("file" + ext) {
			t.Errorf("expected '%s' to be allowed", ext)
		}
	}
}

func TestHasAllowedExtension_UnknownExtension(t *testing.T) {
	if packagekit.HasAllowedExtension("malware.exe") {
		t.Error("'.exe' should NOT be allowed")
	}
	if packagekit.HasAllowedExtension("script.bin") {
		t.Error("'.bin' should NOT be allowed")
	}
	if packagekit.HasAllowedExtension("empty") {
		t.Error("no-extension file should NOT be allowed")
	}
}

// ============================================================================
// Content signature tests
// ============================================================================

func TestValidateContent_PNG(t *testing.T) {
	pngHeader := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00}
	err := packagekit.ValidateContentMatchesExtension("icon.png", pngHeader)
	if err != "" {
		t.Errorf("valid PNG should pass, got: %s", err)
	}

	err = packagekit.ValidateContentMatchesExtension("icon.png", []byte{0x00, 0x01})
	if err == "" {
		t.Error("invalid PNG should fail")
	}
}

func TestValidateContent_JPEG(t *testing.T) {
	jpegHeader := []byte{0xff, 0xd8, 0xff, 0xe0}
	err := packagekit.ValidateContentMatchesExtension("photo.jpg", jpegHeader)
	if err != "" {
		t.Errorf("valid JPEG should pass, got: %s", err)
	}
	jpegHeader = []byte{0xff, 0xd8, 0xff, 0x00}
	err = packagekit.ValidateContentMatchesExtension("photo.jpeg", jpegHeader)
	if err != "" {
		t.Errorf("valid JPEG (jpeg extension) should pass, got: %s", err)
	}
}

func TestValidateContent_GIF(t *testing.T) {
	err := packagekit.ValidateContentMatchesExtension("anim.gif", []byte("GIF89a"))
	if err != "" {
		t.Errorf("valid GIF should pass, got: %s", err)
	}
}

func TestValidateContent_PDF(t *testing.T) {
	err := packagekit.ValidateContentMatchesExtension("doc.pdf", []byte("%PDF-1.4"))
	if err != "" {
		t.Errorf("valid PDF should pass, got: %s", err)
	}
}

func TestValidateContent_WebP(t *testing.T) {
	webpHeader := []byte("RIFF\x00\x00\x00\x00WEBP")
	err := packagekit.ValidateContentMatchesExtension("image.webp", webpHeader)
	if err != "" {
		t.Errorf("valid WebP should pass, got: %s", err)
	}
}

func TestValidateContent_ICO(t *testing.T) {
	err := packagekit.ValidateContentMatchesExtension("favicon.ico", []byte{0x00, 0x00, 0x01, 0x00, 0x00})
	if err != "" {
		t.Errorf("valid ICO should pass, got: %s", err)
	}
}

func TestValidateContent_SVG(t *testing.T) {
	err := packagekit.ValidateContentMatchesExtension("icon.svg", []byte("<svg xmlns='...'>"))
	if err != "" {
		t.Errorf("valid SVG should pass, got: %s", err)
	}

	err = packagekit.ValidateContentMatchesExtension("icon.svg", []byte("not an svg"))
	if err == "" {
		t.Error("invalid SVG should fail")
	}
}

func TestValidateContent_UTF8Text(t *testing.T) {
	err := packagekit.ValidateContentMatchesExtension("readme.md", []byte("# Hello World"))
	if err != "" {
		t.Errorf("valid UTF-8 text should pass, got: %s", err)
	}
}

func TestValidateContent_TextWithNullByte(t *testing.T) {
	err := packagekit.ValidateContentMatchesExtension("script.py", []byte{0x00, 0x01, 0x02})
	if err == "" {
		t.Error("text with null byte should fail")
	}
}

// ============================================================================
// SKILL.md parser tests
// ============================================================================

func TestParse_MinimalValid(t *testing.T) {
	parser := packagekit.NewSkillMetadataParser()
	content := "---\nname: test-skill\ndescription: A test skill\n---\n# Body\n\nHello world"
	meta, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Name != "test-skill" {
		t.Errorf("expected name 'test-skill', got '%s'", meta.Name)
	}
	if meta.Description != "A test skill" {
		t.Errorf("expected description 'A test skill', got '%s'", meta.Description)
	}
	if meta.Version != "" {
		t.Errorf("expected empty version, got '%s'", meta.Version)
	}
	if !strings.Contains(meta.Body, "Hello world") {
		t.Errorf("expected body to contain 'Hello world', got '%s'", meta.Body)
	}
}

func TestParse_WithVersion(t *testing.T) {
	parser := packagekit.NewSkillMetadataParser()
	content := "---\nname: my-skill\ndescription: desc\nversion: 1.0.0\n---\n# Body"
	meta, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", meta.Version)
	}
}

func TestParse_NestedMetadataVersion(t *testing.T) {
	parser := packagekit.NewSkillMetadataParser()
	content := "---\nname: my-skill\ndescription: desc\nmetadata:\n  version: 2.0.0\n---\n# Body"
	meta, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Version != "2.0.0" {
		t.Errorf("expected version '2.0.0', got '%s'", meta.Version)
	}
}

func TestParse_MissingName(t *testing.T) {
	parser := packagekit.NewSkillMetadataParser()
	content := "---\ndescription: desc\n---\n# Body"
	_, err := parser.Parse(content)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParse_MissingFrontmatterStart(t *testing.T) {
	parser := packagekit.NewSkillMetadataParser()
	_, err := parser.Parse("no frontmatter here")
	if err == nil {
		t.Fatal("expected error for missing frontmatter")
	}
}

func TestParse_MissingClosingDelimiter(t *testing.T) {
	parser := packagekit.NewSkillMetadataParser()
	content := "---\nname: test\ndescription: desc\n# No closing delimiter"
	_, err := parser.Parse(content)
	if err == nil {
		t.Fatal("expected error for missing closing delimiter")
	}
}

func TestParse_EmptyContent(t *testing.T) {
	parser := packagekit.NewSkillMetadataParser()
	_, err := parser.Parse("")
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestParse_FrontmatterWithQuotedValues(t *testing.T) {
	parser := packagekit.NewSkillMetadataParser()
	content := `---
name: "quoted-name"
description: 'quoted desc'
version: "1.0.0"
---
# Body`
	meta, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Name != "quoted-name" {
		t.Errorf("expected name 'quoted-name', got '%s'", meta.Name)
	}
}

func TestParse_LooseFallback(t *testing.T) {
	parser := packagekit.NewSkillMetadataParser()
	// Content that would fail strict YAML parsing but has key:value lines.
	content := "---\nname: loose-skill\ndescription: A loose parse skill\n---\n# Body"
	meta, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Name != "loose-skill" {
		t.Errorf("expected name 'loose-skill', got '%s'", meta.Name)
	}
}

// ============================================================================
// Package validator tests
// ============================================================================

func TestValidator_SKILLMDRequired(t *testing.T) {
	v := packagekit.NewSkillPackageValidator(nil)
	result := v.Validate([]packagekit.PackageEntry{
		{Path: "README.md", Content: []byte("# Hello"), Size: 7, ContentType: "text/markdown"},
	})
	if result.Passed() {
		t.Fatal("expected validation failure without SKILL.md")
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected at least one error")
	}
}

func TestValidator_ValidPackage(t *testing.T) {
	v := packagekit.NewSkillPackageValidator(nil)
	skillMd := "---\nname: test\ndescription: A test\n---\n# Body"
	result := v.Validate([]packagekit.PackageEntry{
		{Path: "SKILL.md", Content: []byte(skillMd), Size: int64(len(skillMd)), ContentType: "text/markdown"},
		{Path: "main.py", Content: []byte("print('hello')"), Size: 14, ContentType: "text/x-python"},
	})
	if !result.Passed() {
		t.Fatalf("expected pass, got errors: %v", result.Errors)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(result.Errors))
	}
}

func TestValidator_ExtensionWarning(t *testing.T) {
	v := packagekit.NewSkillPackageValidator(nil)
	skillMd := "---\nname: test\ndescription: A test\n---\n# Body"
	result := v.Validate([]packagekit.PackageEntry{
		{Path: "SKILL.md", Content: []byte(skillMd), Size: int64(len(skillMd)), ContentType: "text/markdown"},
		{Path: "data.bin", Content: []byte{0x01, 0x02}, Size: 2, ContentType: "application/octet-stream"},
	})
	if !result.Passed() {
		t.Fatalf("expected pass (warnings are not errors), got errors: %v", result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning for unknown extension")
	}
}

func TestValidator_TooManyFiles(t *testing.T) {
	v := packagekit.NewSkillPackageValidator(nil)
	skillMd := "---\nname: test\ndescription: desc\n---\n# Body"
	entries := []packagekit.PackageEntry{
		{Path: "SKILL.md", Content: []byte(skillMd), Size: int64(len(skillMd)), ContentType: "text/markdown"},
	}
	// Add 501 entries (exceeds 500).
	for i := 0; i < 500; i++ {
		entries = append(entries, packagekit.PackageEntry{
			Path: "file.txt", Content: []byte("x"), Size: 1, ContentType: "text/plain",
		})
	}
	if len(entries) != 501 {
		t.Fatalf("expected 501 entries, got %d", len(entries))
	}
	result := v.Validate(entries)
	if result.Passed() {
		t.Fatal("expected failure for too many files")
	}
}

func TestValidator_LargeFile(t *testing.T) {
	v := packagekit.NewSkillPackageValidator(nil)
	skillMd := "---\nname: test\ndescription: desc\n---\n# Body"
	largeContent := make([]byte, 11*1024*1024) // 11 MB > 10 MB limit
	result := v.Validate([]packagekit.PackageEntry{
		{Path: "SKILL.md", Content: []byte(skillMd), Size: int64(len(skillMd)), ContentType: "text/markdown"},
		{Path: "large.bin", Content: largeContent, Size: int64(len(largeContent)), ContentType: "application/octet-stream"},
	})
	if result.Passed() {
		t.Fatal("expected failure for large file")
	}
}

func TestValidator_DuplicatePath(t *testing.T) {
	v := packagekit.NewSkillPackageValidator(nil)
	skillMd := "---\nname: test\ndescription: desc\n---\n# Body"
	result := v.Validate([]packagekit.PackageEntry{
		{Path: "SKILL.md", Content: []byte(skillMd), Size: int64(len(skillMd)), ContentType: "text/markdown"},
		{Path: "README.md", Content: []byte("hello"), Size: 5, ContentType: "text/markdown"},
		{Path: "README.md", Content: []byte("world"), Size: 5, ContentType: "text/markdown"},
	})
	if result.Passed() {
		t.Fatal("expected failure for duplicate path")
	}
}

func TestValidator_ContentMismatchWarning(t *testing.T) {
	v := packagekit.NewSkillPackageValidator(nil)
	skillMd := "---\nname: test\ndescription: desc\n---\n# Body"
	result := v.Validate([]packagekit.PackageEntry{
		{Path: "SKILL.md", Content: []byte(skillMd), Size: int64(len(skillMd)), ContentType: "text/markdown"},
		{Path: "image.png", Content: []byte("not a png"), Size: 10, ContentType: "image/png"},
	})
	if !result.Passed() {
		t.Fatalf("expected pass (content mismatch is a warning), got errors: %v", result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning for content mismatch")
	}
}

var _ = strings.Contains
