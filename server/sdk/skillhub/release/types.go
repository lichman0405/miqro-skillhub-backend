// Package release provides first-class skill release objects on top of
// published skill versions.
//
// A Release carries distribution metadata (title, notes, channel, draft,
// prerelease flags), provenance (publisher, reviewer, package hash, CI
// check-run IDs), and optional assets.
package release

import "time"

// Release represents a published skill version release.
// Each published version can have exactly one stable release per channel.
type Release struct {
	ID           int64
	SkillID      int64
	VersionID    int64
	Channel      string // "stable", "beta", etc.
	Title        string
	Notes        string // release notes / changelog
	Draft        bool
	Prerelease   bool
	Yanked       bool
	PublishedAt  *time.Time
	PublisherID  string
	ReviewerID   *string
	PackageHash  *string  // SHA-256 of the published package bundle
	CiCheckRunID *string  // Phase 12 nullable field
	MetadataJSON *string  // jsonb — extensible metadata
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ReleaseAsset represents a downloadable asset attached to a release.
type ReleaseAsset struct {
	ID            int64
	ReleaseID     int64
	Name          string // filename
	Label         *string
	ContentType   string
	Size          int64
	StorageKey    string
	SHA256        *string
	DownloadCount int64
	CreatedAt     time.Time
}

// Provenance carries the release provenance metadata.
type Provenance struct {
	PublisherID  string  `json:"publisherId"`
	ReviewerID   *string `json:"reviewerId,omitempty"`
	PackageHash  *string `json:"packageHash,omitempty"`
	CiCheckRunID *string `json:"ciCheckRunId,omitempty"`
}
