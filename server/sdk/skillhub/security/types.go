package security

import "time"

// SecurityAudit records security scan results for a skill version.
type SecurityAudit struct {
	ID                  int64
	SkillVersionID      int64
	ScanID              *string
	ScannerType         string // skill-scanner, custom
	Verdict             string // SAFE, SUSPICIOUS, DANGEROUS, BLOCKED
	IsSafe              bool
	MaxSeverity         *string
	FindingsCount       int
	Findings            string // jsonb, default '[]'
	ScanDurationSeconds *float64
	ScannedAt           *time.Time
	CreatedAt           time.Time
	DeletedAt           *time.Time // soft delete
}

// Scanner type constants.
const (
	ScannerTypeSkillScanner = "skill-scanner"
	ScannerTypeCustom       = "custom"
)

// Verdict constants.
const (
	VerdictSafe      = "SAFE"
	VerdictSuspicious = "SUSPICIOUS"
	VerdictDangerous = "DANGEROUS"
	VerdictBlocked   = "BLOCKED"
)
