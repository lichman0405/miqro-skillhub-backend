package packageupload

import (
	"archive/zip"
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
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

// makeMultipartRequest creates a POST request with a "package" file field
// containing buf.
func makeMultipartRequest(buf []byte) *http.Request {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("package", "test.zip")
	fw.Write(buf)
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

// ── ExtractZip tests ──────────────────────────────────────────────────────────

func TestExtractZip_NormalSmallPackage(t *testing.T) {
	zipBytes := buildZip(map[string][]byte{
		"SKILL.md":  []byte("---\nname: test\n---\nbody"),
		"README.md": []byte("# Hello"),
		"a.txt":     []byte("hello world"),
	})

	entries, err := ExtractZip(zipBytes)
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
	n := packagekit.MaxFileCount + 1
	files := make(map[string][]byte, n)
	for i := 0; i < n; i++ {
		name := "f" + strings.Repeat("x", 8-len(itoa(i))) + itoa(i) + ".txt"
		files[name] = []byte("a")
	}
	zipBytes := buildZip(files)

	_, err := ExtractZip(zipBytes)
	if err == nil {
		t.Fatal("expected error for too many entries, got nil")
	}
	if !strings.Contains(err.Error(), "contains") {
		t.Errorf("error should mention entry count, got: %v", err)
	}
}

func TestExtractZip_SingleEntryExceedsMaxSize(t *testing.T) {
	content := make([]byte, packagekit.MaxSingleFileSize+1)
	for i := range content {
		content[i] = 'x'
	}
	zipBytes := buildZip(map[string][]byte{
		"SKILL.md": []byte("---\nname: test\n---\n"),
		"big.txt":  content,
	})

	_, err := ExtractZip(zipBytes)
	if err == nil {
		t.Fatal("expected error for oversized entry, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("error should mention size exceed, got: %v", err)
	}
}

func TestExtractZip_TotalDecompressedExceedsMax(t *testing.T) {
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

	_, err := ExtractZip(zipBytes)
	if err == nil {
		t.Fatal("expected error for total size exceeded, got nil")
	}
	if !strings.Contains(err.Error(), "total decompressed size") {
		t.Errorf("error should mention total decompressed size, got: %v", err)
	}
}

func TestExtractZip_ZipSlipRejected(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("../etc/passwd")
	fw.Write([]byte("malicious"))
	w.Close()

	_, err := ExtractZip(buf.Bytes())
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

	entries, err := ExtractZip(buf.Bytes())
	if err != nil {
		t.Fatalf("unexpected error for empty zip: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries from empty zip, got %d", len(entries))
	}
}

func TestExtractZip_InvalidZipBytes(t *testing.T) {
	_, err := ExtractZip([]byte("not a zip file"))
	if err == nil {
		t.Fatal("expected error for invalid zip bytes")
	}
	if !strings.Contains(err.Error(), "invalid zip") {
		t.Errorf("error should mention invalid zip, got: %v", err)
	}
}

func TestExtractZip_DirectorySkipped(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("SKILL.md")
	fw.Write([]byte("---\nname: test\n---\n"))
	w.Create("docs/")
	w.Close()

	entries, err := ExtractZip(buf.Bytes())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry (directory skipped), got %d", len(entries))
	}
}

func TestExtractZip_EntryLargerThanHeaderClaims(t *testing.T) {
	// Create a zip entry whose actual content is larger than the header claims
	// when the compressed form happens to be close to the real size.
	content := make([]byte, packagekit.MaxSingleFileSize+1)
	for i := range content {
		content[i] = 'x'
	}
	zipBytes := buildZip(map[string][]byte{
		"oversized.bin": content,
	})

	_, err := ExtractZip(zipBytes)
	if err == nil {
		t.Fatal("expected error for entry larger than MaxSingleFileSize, got nil")
	}
}

// ── ReadPackageFromRequest tests ─────────────────────────────────────────────

func TestReadPackageFromRequest_NormalUpload(t *testing.T) {
	zipBytes := buildZip(map[string][]byte{
		"SKILL.md": []byte("---\nname: test\n---\nbody"),
	})
	req := makeMultipartRequest(zipBytes)

	entries, err := ReadPackageFromRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

// buildZipStored creates an in-memory zip using zip.Store (no compression)
// so the on-disk size equals the uncompressed content size plus headers.
func buildZipStored(files map[string][]byte) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		fw, _ := w.CreateHeader(&zip.FileHeader{
			Name:   name,
			Method: zip.Store,
		})
		_, _ = fw.Write(content)
	}
	w.Close()
	return buf.Bytes()
}

func TestReadPackageFromRequest_UploadExceedsMaxSize(t *testing.T) {
	// Use Store (no compression) with many small files — each under the
	// single-file limit — so the on-disk zip exceeds MaxUploadZipSize.
	smallSize := 64 * 1024 // 64KB per file
	needFiles := (packagekit.MaxUploadZipSize / smallSize) + 5

	files := make(map[string][]byte, needFiles+1)
	files["SKILL.md"] = []byte("---\nname: test\n---\n")
	for i := 0; i < needFiles; i++ {
		content := make([]byte, smallSize)
		for j := range content {
			content[j] = byte(i + j)
		}
		name := "f" + strings.Repeat("0", 8-len(itoa(i))) + itoa(i) + ".bin"
		files[name] = content
	}
	zipBytes := buildZipStored(files)

	if int64(len(zipBytes)) <= packagekit.MaxUploadZipSize {
		t.Fatalf("expected zip > %d bytes, got %d", packagekit.MaxUploadZipSize, len(zipBytes))
	}

	req := makeMultipartRequest(zipBytes)

	_, err := ReadPackageFromRequest(req)
	if err == nil {
		t.Fatal("expected error for upload exceeding max size, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("error should mention exceeds maximum size, got: %v", err)
	}
}

func TestReadPackageFromRequest_NoPackageField(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=foo")
	_, err := ReadPackageFromRequest(req)
	if err == nil {
		t.Fatal("expected error for missing package field, got nil")
	}
}

func TestReadPackageFromRequest_EmptyPackage(t *testing.T) {
	req := makeMultipartRequest([]byte{})
	_, err := ReadPackageFromRequest(req)
	if err == nil {
		t.Fatal("expected error for empty package (invalid zip), got nil")
	}
	if !strings.Contains(err.Error(), "invalid zip") {
		t.Errorf("error should mention invalid zip, got: %v", err)
	}
}

func TestReadPackageFromRequest_ZipSlipInUpload(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("../etc/passwd")
	fw.Write([]byte("malicious"))
	w.Close()

	req := makeMultipartRequest(buf.Bytes())

	_, err := ReadPackageFromRequest(req)
	if err == nil {
		t.Fatal("expected error for zip-slip in upload, got nil")
	}
	if !strings.Contains(err.Error(), "insecure") {
		t.Errorf("error should mention insecure path, got: %v", err)
	}
}

func TestReadPackageFromRequest_TooManyEntries(t *testing.T) {
	n := packagekit.MaxFileCount + 1
	files := make(map[string][]byte, n)
	for i := 0; i < n; i++ {
		name := "f" + strings.Repeat("0", 8-len(itoa(i))) + itoa(i) + ".txt"
		files[name] = []byte("a")
	}
	zipBytes := buildZip(files)
	req := makeMultipartRequest(zipBytes)

	_, err := ReadPackageFromRequest(req)
	if err == nil {
		t.Fatal("expected error for too many entries via upload, got nil")
	}
}

func TestReadPackageFromRequest_TotalDecompressedExceedsMax(t *testing.T) {
	perFile := packagekit.MaxTotalPackageSize/250 + 1
	n := 251
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
	req := makeMultipartRequest(zipBytes)

	_, err := ReadPackageFromRequest(req)
	if err == nil {
		t.Fatal("expected error for total decompressed exceeded via upload, got nil")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

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

// Ensure io is used (imported for multipart request body).
var _ = io.EOF
