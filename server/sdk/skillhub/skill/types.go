package skill

import "time"

// Skill represents a logical skill line.
type Skill struct {
	ID               int64
	NamespaceID      int64
	Slug             string
	DisplayName      string
	Summary          string // TEXT
	OwnerID          string
	SourceSkillID    *int64
	Visibility       string // PUBLIC, NAMESPACE_ONLY, PRIVATE
	Status           string // ACTIVE, HIDDEN, ARCHIVED
	LatestVersionID  *int64
	DownloadCount    int64
	StarCount        int
	RatingAvg        float64 // DECIMAL(3,2)
	RatingCount      int
	SubscriptionCount int
	Hidden           bool
	HiddenAt         *time.Time
	HiddenBy         *string
	CreatedBy        *string
	CreatedAt        time.Time
	UpdatedBy        *string
	UpdatedAt        time.Time
}

// SkillVersion represents an immutable uploaded/released version.
type SkillVersion struct {
	ID                  int64
	SkillID             int64
	Version             string
	Status              string // DRAFT, SCANNING, SCAN_FAILED, UPLOADED, PENDING_REVIEW, PUBLISHED, REJECTED, YANKED
	Changelog           string
	ParsedMetadataJSON  *string // jsonb
	ManifestJSON        *string // jsonb
	RequestedVisibility *string
	FileCount           int
	TotalSize           int64
	BundleReady         bool
	DownloadReady       bool
	PublishedAt         *time.Time
	YankedAt            *time.Time
	YankedBy            *string
	YankReason          *string
	CreatedBy           string
	CreatedAt           time.Time
}

// SkillFile represents stored file metadata for a version.
type SkillFile struct {
	ID          int64
	VersionID   int64
	FilePath    string
	FileSize    int64
	ContentType string
	SHA256      string
	StorageKey  string
	CreatedAt   time.Time
}

// SkillTag represents a tag on a skill.
type SkillTag struct {
	ID        int64
	SkillID   int64
	TagName   string
	VersionID int64
	CreatedBy *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SkillVersionStats tracks per-version download stats.
type SkillVersionStats struct {
	SkillVersionID int64
	SkillID        int64
	DownloadCount  int64
	UpdatedAt      time.Time
}

// SkillStorageDeletionCompensation records storage objects pending deletion.
type SkillStorageDeletionCompensation struct {
	ID              int64
	SkillID         *int64
	Namespace       string
	Slug            string
	StorageKeysJSON string // TEXT
	Status          string // PENDING, COMPLETED
	AttemptCount    int
	LastError       *string
	LastAttemptAt   *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// SkillSearchDocument is the denormalized search index entry.
type SkillSearchDocument struct {
	ID             int64
	SkillID        int64
	NamespaceID    int64
	NamespaceSlug  string
	OwnerID        string
	Title          string
	Summary        string
	Keywords       string
	SearchText     string
	SemanticVector *string
	Visibility     string
	Status         string
	UpdatedAt      time.Time
}
