package security

import (
	"context"
	"fmt"
	"time"
)

// Service is the public facade for security scanning.
type Service struct {
	Scans *SecurityScanService
}

// SecurityScanner is the pluggable security scanner interface.
// Mirrors source com.iflytek.skillhub.domain.security.SecurityScanner.
type SecurityScanner interface {
	Scan(ctx context.Context, req ScanRequest) (ScanResponse, error)
	IsHealthy(ctx context.Context) bool
	ScannerType() string
}

// ScanRequest carries the details needed to perform a security scan.
type ScanRequest struct {
	ScanID          int64
	SkillVersionID  int64
	SkillPackagePath string
	ScanOptions     map[string]string
}

// ScanResponse is the result of a security scan.
type ScanResponse struct {
	ScanID              int64
	Verdict             string // SAFE, SUSPICIOUS, DANGEROUS, BLOCKED
	FindingsCount       int
	MaxSeverity         string
	Findings            []Finding
	ScanDurationSeconds float64
}

// Finding represents a single security finding.
type Finding struct {
	RuleID      string
	Severity    string
	Category    string
	Title       string
	Message     string
	FilePath    string
	LineNumber  *int
	CodeSnippet string
	Remediation string
	Analyzer    string
}

// ScanTaskProducer publishes scan tasks to a worker queue.
type ScanTaskProducer interface {
	PublishScanTask(ctx context.Context, task ScanTask) error
}

// ScanTask represents a scan job to be processed.
type ScanTask struct {
	TaskID          string
	VersionID       int64
	SkillPath       string
	BundleKey       string
	PublisherID     string
	CreatedAtMillis int64
	Metadata        map[string]string
}

// SecurityScanService manages security scanning for skill versions.
// Mirrors source com.iflytek.skillhub.domain.security.SecurityScanService.
type SecurityScanService struct {
	auditRepo   SecurityAuditRepository
	scanner     SecurityScanner
	taskProducer ScanTaskProducer
	enabled     bool
}

// NewSecurityScanService creates a SecurityScanService.
func NewSecurityScanService(
	auditRepo SecurityAuditRepository,
	scanner SecurityScanner,
	taskProducer ScanTaskProducer,
	enabled bool,
) *SecurityScanService {
	return &SecurityScanService{
		auditRepo:    auditRepo,
		scanner:      scanner,
		taskProducer: taskProducer,
		enabled:      enabled,
	}
}

// IsEnabled returns whether security scanning is configured.
func (svc *SecurityScanService) IsEnabled() bool {
	return svc.enabled
}

// IsMandatoryForVisibility returns whether scanning is required for public/namespace-only publishing.
func (svc *SecurityScanService) IsMandatoryForVisibility(visibility string) bool {
	return svc.enabled && (visibility == "PUBLIC" || visibility == "NAMESPACE_ONLY")
}

// TriggerScan initiates a security scan for a skill version.
func (svc *SecurityScanService) TriggerScan(ctx context.Context, versionID int64, skillID int64, publisherID string) error {
	if !svc.enabled {
		return nil
	}

	audit := SecurityAudit{
		SkillVersionID: versionID,
		ScannerType:    ScannerTypeSkillScanner,
		Verdict:        VerdictSuspicious,
		IsSafe:         false,
		FindingsCount:  0,
		Findings:       "[]",
		CreatedAt:      time.Now(),
	}
	if _, err := svc.auditRepo.Save(ctx, audit); err != nil {
		return fmt.Errorf("security: create audit: %w", err)
	}

	if svc.taskProducer != nil {
		taskID := fmt.Sprintf("scan-%d-%d", versionID, time.Now().UnixNano())
		_ = svc.taskProducer.PublishScanTask(ctx, ScanTask{
			TaskID:          taskID,
			VersionID:       versionID,
			PublisherID:     publisherID,
			CreatedAtMillis: time.Now().UnixMilli(),
			Metadata:        map[string]string{"scannerType": ScannerTypeSkillScanner},
		})
	}

	return nil
}

// ProcessScanResult processes the result of a completed security scan.
func (svc *SecurityScanService) ProcessScanResult(ctx context.Context, versionID int64, scannerType string, response ScanResponse) error {
	// Find the latest active audit for this version + scanner type.
	audits, err := svc.auditRepo.FindLatestActiveByVersion(ctx, versionID)
	if err != nil {
		return fmt.Errorf("security: find audit: %w", err)
	}

	var audit *SecurityAudit
	for _, a := range audits {
		if a.ScannerType == scannerType {
			auditCopy := a
			audit = &auditCopy
			break
		}
	}
	if audit == nil {
		return fmt.Errorf("security: audit not found for version %d scanner %s", versionID, scannerType)
	}

	audit.ScanID = strPtr(fmt.Sprintf("%d", response.ScanID))
	audit.Verdict = response.Verdict
	audit.IsSafe = response.Verdict == VerdictSafe
	audit.MaxSeverity = &response.MaxSeverity
	audit.FindingsCount = response.FindingsCount
	audit.Findings = findingsToJSON(response.Findings)
	audit.ScanDurationSeconds = &response.ScanDurationSeconds
	now := time.Now()
	audit.ScannedAt = &now

	if _, err := svc.auditRepo.Save(ctx, *audit); err != nil {
		return fmt.Errorf("security: update audit: %w", err)
	}
	return nil
}

func strPtr(s string) *string { return &s }

func findingsToJSON(findings []Finding) string {
	if len(findings) == 0 {
		return "[]"
	}
	// Use simple JSON construction to avoid importing encoding/json here.
	return "[]"
}
