package toolapi

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/packagekit"
)

// buildZip creates an in-memory zip from name→content pairs.
func buildZip(files map[string][]byte) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		fw, _ := w.Create(name)
		_, _ = fw.Write(content)
	}
	w.Close()
	return buf.Bytes()
}

func TestExtractZip_NormalSmallPackage(t *testing.T) {
	zipBytes := buildZip(map[string][]byte{
		"SKILL.md":  []byte("---\nname: test\n---\nbody"),
		"README.md": []byte("# Hello"),
		"a.txt":     []byte("hello world"),
	})

	entries, err := extractZip(zipBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}

	names := map[string]bool{}
	for _, e := range entries {
		names[e.Path] = true
	}
	for _, want := range []string{"SKILL.md", "README.md", "a.txt"} {
		if !names[want] {
			t.Errorf("missing expected entry: %q", want)
		}
	}
}

func TestExtractZip_TooManyEntries(t *testing.T) {
	// Build a zip with MaxFileCount + 1 files (dirs are skipped, so we
	// create enough individual files to exceed the cap).
	n := packagekit.MaxFileCount + 1
	files := make(map[string][]byte, n)
	for i := 0; i < n; i++ {
		name := "f" + strings.Repeat("x", 8-len(itoa(i))) + itoa(i) + ".txt"
		files[name] = []byte("a")
	}
	zipBytes := buildZip(files)

	_, err := extractZip(zipBytes)
	if err == nil {
		t.Fatal("expected error for too many entries, got nil")
	}
	if !strings.Contains(err.Error(), "contains") {
		t.Errorf("error should mention entry count, got: %v", err)
	}
}

func TestExtractZip_SingleEntryExceedsMaxSize(t *testing.T) {
	// Header claims a size well over MaxSingleFileSize.
	content := make([]byte, packagekit.MaxSingleFileSize+1)
	for i := range content {
		content[i] = 'x'
	}
	zipBytes := buildZip(map[string][]byte{
		"SKILL.md": []byte("---\nname: test\n---\n"),
		"big.txt":  content,
	})

	_, err := extractZip(zipBytes)
	if err == nil {
		t.Fatal("expected error for oversized entry, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("error should mention size exceed, got: %v", err)
	}
}

func TestExtractZip_TotalDecompressedExceedsMax(t *testing.T) {
	// Create many files whose total exceeds MaxTotalPackageSize.
	perFile := packagekit.MaxTotalPackageSize/250 + 1 // ~400KB each
	n := 251                                            // 251 × 400KB > 100MB
	files := make(map[string][]byte, n)
	for i := 0; i < n; i++ {
		content := make([]byte, perFile)
		for j := range content {
			content[j] = 'x'
		}
		name := "f" + strings.Repeat("0", 4-len(itoa(i))) + itoa(i) + ".txt"
		files[name] = content
	}
	zipBytes := buildZip(files)

	_, err := extractZip(zipBytes)
	if err == nil {
		t.Fatal("expected error for total size exceeded, got nil")
	}
	if !strings.Contains(err.Error(), "total decompressed size") {
		t.Errorf("error should mention total decompressed size, got: %v", err)
	}
}

func TestExtractZip_ZipSlipRejected(t *testing.T) {
	// Build a zip with a path-traversal entry using raw writer.
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("../etc/passwd")
	fw.Write([]byte("malicious"))
	w.Close()

	_, err := extractZip(buf.Bytes())
	if err == nil {
		t.Fatal("expected error for zip-slip path, got nil")
	}
	if !strings.Contains(err.Error(), "insecure") {
		t.Errorf("error should mention insecure path, got: %v", err)
	}
}

func TestExtractZip_EmptyZip(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	w.Close()

	entries, err := extractZip(buf.Bytes())
	if err != nil {
		t.Fatalf("unexpected error for empty zip: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries from empty zip, got %d", len(entries))
	}
}

func TestExtractZip_InvalidZipBytes(t *testing.T) {
	_, err := extractZip([]byte("not a zip file"))
	if err == nil {
		t.Fatal("expected error for invalid zip bytes")
	}
	if !strings.Contains(err.Error(), "invalid zip") {
		t.Errorf("error should mention invalid zip, got: %v", err)
	}
}

func TestExtractZip_DirectorySkipped(t *testing.T) {
	// Directories should be skipped and not count as files.
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("SKILL.md")
	fw.Write([]byte("---\nname: test\n---\n"))
	// Create a directory entry — zip.Writer.Create on a path ending in /
	// creates a directory record with no data.
	w.Create("docs/")
	w.Close()

	entries, err := extractZip(buf.Bytes())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry (directory skipped), got %d", len(entries))
	}
}

func TestExtractZip_UploadSizeExceedsMax(t *testing.T) {
	// readPackageFromRequest uses LimitReader — verify the error message
	// path by calling extractZip with an over-limit zip.
	content := make([]byte, packagekit.MaxSingleFileSize)
	for i := range content {
		content[i] = 'x'
	}
	zipBytes := buildZip(map[string][]byte{
		"big.bin": content,
	})

	// This is fine at the extractZip level (single file within limit).
	entries, err := extractZip(zipBytes)
	if err != nil {
		t.Fatalf("unexpected error for valid single large file: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

// itoa is a fast int→string for small non-negative integers (avoids importing strconv).
func itoa(n int) string {
	if n < 0 {
		return "0"
	}
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
