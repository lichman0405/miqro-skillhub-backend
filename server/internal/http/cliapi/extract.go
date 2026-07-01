package cliapi

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	"miqro-skillhub/server/sdk/skillhub/packagekit"
)

// init wires the shared zip extraction to the real implementation so that
// both CLI and portal handlers can decompress package uploads without an
// import cycle.
func init() {
	extractZipBytes = extractZipArchive
}

func extractZipArchive(src []byte) ([]packagekit.PackageEntry, error) {
	zr, err := zip.NewReader(bytes.NewReader(src), int64(len(src)))
	if err != nil {
		return nil, err
	}
	var entries []packagekit.PackageEntry
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		// Reject zip-slip paths.
		if !filepath.IsLocal(f.Name) {
			return nil, fmt.Errorf("cliapi: insecure zip entry path %q", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
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
