// Package packageupload provides bounded zip extraction shared by the CLI
// API, tool API, and portal HTTP handlers.  Every public function enforces
// the package policy constants from packagekit — upload size, entry count,
// single-entry decompressed size, total decompressed size, and zip-slip
// rejection — so that no handler needs to duplicate those checks.
package packageupload

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	"miqro-skillhub/server/sdk/skillhub/packagekit"
)

// ReadPackageFromRequest reads a multipart "package" field from r, bounds the
// upload to packagekit.MaxUploadZipSize, and extracts the zip with all policy
// limits enforced.  Callers receive a validated []PackageEntry ready for the
// SDK publish / validate services.
func ReadPackageFromRequest(r *http.Request) ([]packagekit.PackageEntry, error) {
	file, _, err := r.FormFile("package")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Cap total upload bytes.  If the zip exceeds MaxUploadZipSize the
	// LimitReader returns io.ErrUnexpectedEOF — we map that to a clear
	// user-facing message.
	body, err := io.ReadAll(io.LimitReader(file, packagekit.MaxUploadZipSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read uploaded zip: %w", err)
	}
	if int64(len(body)) > packagekit.MaxUploadZipSize {
		return nil, fmt.Errorf("uploaded zip exceeds maximum size of %d bytes", packagekit.MaxUploadZipSize)
	}

	return ExtractZip(body)
}

// ExtractZip decompresses a zip byte slice into PackageEntry values, bounded
// by the package policy constants:
//   - MaxFileCount     — reject zips with too many entries
//   - MaxSingleFileSize — reject individual entries that are too large
//   - MaxTotalPackageSize — reject zips whose total decompressed size is too large
//   - zip-slip rejection via filepath.IsLocal
func ExtractZip(src []byte) ([]packagekit.PackageEntry, error) {
	zr, err := zip.NewReader(bytes.NewReader(src), int64(len(src)))
	if err != nil {
		return nil, fmt.Errorf("invalid zip: %w", err)
	}

	if len(zr.File) > packagekit.MaxFileCount {
		return nil, fmt.Errorf(
			"zip contains %d entries (max %d)", len(zr.File), packagekit.MaxFileCount,
		)
	}

	var entries []packagekit.PackageEntry
	var totalSize int64
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		// Reject zip-slip paths.
		if !filepath.IsLocal(f.Name) {
			return nil, fmt.Errorf("insecure zip entry path %q", f.Name)
		}

		if f.UncompressedSize64 > uint64(packagekit.MaxSingleFileSize) {
			return nil, fmt.Errorf(
				"zip entry %q decompressed size %d exceeds max %d bytes",
				f.Name, f.UncompressedSize64, packagekit.MaxSingleFileSize,
			)
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open zip entry %q: %w", f.Name, err)
		}
		// Cap single-entry read to MaxSingleFileSize + 1 so we can detect
		// entries whose header understates the real size.
		content, err := io.ReadAll(io.LimitReader(rc, int64(packagekit.MaxSingleFileSize)+1))
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read zip entry %q: %w", f.Name, err)
		}
		if int64(len(content)) > int64(packagekit.MaxSingleFileSize) {
			return nil, fmt.Errorf(
				"zip entry %q exceeds max single file size of %d bytes",
				f.Name, packagekit.MaxSingleFileSize,
			)
		}
		if len(content) == 0 && f.UncompressedSize64 > 0 {
			return nil, fmt.Errorf("zip entry %q is empty after decompression", f.Name)
		}

		totalSize += int64(len(content))
		if totalSize > int64(packagekit.MaxTotalPackageSize) {
			return nil, fmt.Errorf(
				"zip total decompressed size exceeds max %d bytes",
				packagekit.MaxTotalPackageSize,
			)
		}

		entries = append(entries, packagekit.PackageEntry{
			Path:        f.Name,
			Content:     content,
			Size:        int64(len(content)),
			ContentType: "application/octet-stream",
		})
	}
	return entries, nil
}
