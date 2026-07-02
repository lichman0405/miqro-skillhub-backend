package cliapi

import (
	"miqro-skillhub/server/internal/http/packageupload"
	"miqro-skillhub/server/sdk/skillhub/packagekit"
)

// extractZipEntries extracts a zip byte slice into PackageEntry values using
// the shared bounded extractor (enforces MaxFileCount, MaxSingleFileSize,
// MaxTotalPackageSize, and zip-slip rejection).
func extractZipEntries(src []byte) ([]packagekit.PackageEntry, error) {
	return packageupload.ExtractZip(src)
}
