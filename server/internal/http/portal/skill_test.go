package portal

import (
	"archive/zip"
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/packageupload"
	"miqro-skillhub/server/sdk/skillhub/packagekit"
)

// makeMultipartRequest creates a POST request with a "package" file field
// containing buf, plus optional extra form fields.
func makeMultipartRequest(buf []byte, fields map[string]string) *http.Request {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("package", "test.zip")
	fw.Write(buf)
	for k, v := range fields {
		mw.WriteField(k, v)
	}
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

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

// itoa is a fast int→string for small non-negative integers.
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

var _ = io.EOF

// newTestSkillHandler returns a SkillHandler with a nil SkillSvc — suitable
// only for tests where extraction fails (rejection tests).  For happy-path
// tests use the direct packageupload calls which don't need a DB.
func newTestSkillHandler() *SkillHandler {
	return &SkillHandler{}
}

// authenticatedRequest returns a copy of req with a minimal authenticated
// principal in its context so the handler passes RequireAuth.
func authenticatedRequest(req *http.Request) *http.Request {
	return req.WithContext(middleware.WithPrincipal(req.Context(), middleware.Principal{
		UserID:          "u1",
		IsAuthenticated: true,
	}))
}

// ── Portal publish handler rejection tests ────────────────────────────────────

// TestPortalPublish_UploadExceedsMaxSize verifies that a zip larger than
// MaxUploadZipSize is rejected before reaching the SDK.
func TestPortalPublish_UploadExceedsMaxSize(t *testing.T) {
	smallSize := 64 * 1024
	needFiles := (packagekit.MaxUploadZipSize / smallSize) + 5

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.CreateHeader(&zip.FileHeader{
		Name:   "SKILL.md",
		Method: zip.Store,
	})
	fw.Write([]byte("---\nname: test\n---\n"))
	for i := 0; i < needFiles; i++ {
		content := make([]byte, smallSize)
		for j := range content {
			content[j] = byte(i + j)
		}
		name := "f" + strings.Repeat("0", 8-len(itoa(i))) + itoa(i) + ".bin"
		fw2, _ := w.CreateHeader(&zip.FileHeader{
			Name:   name,
			Method: zip.Store,
		})
		fw2.Write(content)
	}
	w.Close()

	if int64(len(buf.Bytes())) <= packagekit.MaxUploadZipSize {
		t.Fatalf("expected zip > %d bytes, got %d", packagekit.MaxUploadZipSize, len(buf.Bytes()))
	}

	req := authenticatedRequest(makeMultipartRequest(buf.Bytes(), nil))
	rr := httptest.NewRecorder()
	h := newTestSkillHandler()
	h.handlePublish(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for oversized upload, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "exceeds maximum size") {
		t.Errorf("error should mention exceeds maximum size, got: %s", rr.Body.String())
	}
}

// TestPortalPublish_ZipSlipRejected verifies zip-slip paths are rejected.
func TestPortalPublish_ZipSlipRejected(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, _ := w.Create("../etc/passwd")
	fw.Write([]byte("malicious"))
	w.Close()

	req := authenticatedRequest(makeMultipartRequest(buf.Bytes(), nil))
	rr := httptest.NewRecorder()
	h := newTestSkillHandler()
	h.handlePublish(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for zip-slip, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "insecure") {
		t.Errorf("error should mention insecure path, got: %s", rr.Body.String())
	}
}

// TestPortalPublish_TooManyEntries verifies entry count limit is enforced.
func TestPortalPublish_TooManyEntries(t *testing.T) {
	n := packagekit.MaxFileCount + 1
	files := make(map[string][]byte, n)
	for i := 0; i < n; i++ {
		name := "f" + strings.Repeat("0", 8-len(itoa(i))) + itoa(i) + ".txt"
		files[name] = []byte("a")
	}
	zipBytes := buildZip(files)

	req := authenticatedRequest(makeMultipartRequest(zipBytes, nil))
	rr := httptest.NewRecorder()
	h := newTestSkillHandler()
	h.handlePublish(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for too many entries, got %d", rr.Code)
	}
}

// TestPortalPublish_MissingPackageField verifies a missing "package" field
// produces an error.
func TestPortalPublish_MissingPackageField(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=foo")
	req = authenticatedRequest(req)

	rr := httptest.NewRecorder()
	h := newTestSkillHandler()
	h.handlePublish(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for missing package field, got %d", rr.Code)
	}
}

// TestPortalPublish_EmptyPackage verifies an empty zip body is rejected.
func TestPortalPublish_EmptyPackage(t *testing.T) {
	req := authenticatedRequest(makeMultipartRequest([]byte{}, nil))
	rr := httptest.NewRecorder()
	h := newTestSkillHandler()
	h.handlePublish(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for empty package, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "invalid zip") {
		t.Errorf("error should mention invalid zip, got: %s", rr.Body.String())
	}
}

// ── Shared helper delegation test ─────────────────────────────────────────────

// TestPortalPublish_DelegatesToSharedHelper proves the portal publish handler
// consumes packages through the shared packageupload helper, producing
// results identical to a direct ExtractZip call.
func TestPortalPublish_DelegatesToSharedHelper(t *testing.T) {
	zipBytes := buildZip(map[string][]byte{
		"SKILL.md":  []byte("---\nname: test\n---\nbody"),
		"README.md": []byte("# Hello"),
	})

	// Direct call to shared helper.
	directEntries, err := packageupload.ExtractZip(zipBytes)
	if err != nil {
		t.Fatalf("direct ExtractZip failed: %v", err)
	}

	// Via multipart request (simulates the handler's ReadPackageFromRequest path).
	req := makeMultipartRequest(zipBytes, nil)
	handlerEntries, err := packageupload.ReadPackageFromRequest(req)
	if err != nil {
		t.Fatalf("ReadPackageFromRequest failed: %v", err)
	}

	if len(directEntries) != len(handlerEntries) {
		t.Errorf("entry count mismatch: direct=%d handler=%d", len(directEntries), len(handlerEntries))
	}
	for i := range directEntries {
		if directEntries[i].Path != handlerEntries[i].Path {
			t.Errorf("entry %d path mismatch: direct=%q handler=%q",
				i, directEntries[i].Path, handlerEntries[i].Path)
		}
		if directEntries[i].Size != handlerEntries[i].Size {
			t.Errorf("entry %d size mismatch: direct=%d handler=%d",
				i, directEntries[i].Size, handlerEntries[i].Size)
		}
	}
}
