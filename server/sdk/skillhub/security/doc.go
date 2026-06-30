// Package security manages scan tasks, scanner interface, audit/finding
// state, and callback handling.
//
// Source module mapping:
//
//	skillhub-domain domain/security
//	  SecurityAudit — security scanning state and findings
//	  ScanTask — scanning task
//	  Scanner interface
//	  Scanner callback handling
//
//	skillhub-infra scanner adapters
//	  No-op scanner for development
//	  External scanner client for production
//
// PUBLIC and NAMESPACE_ONLY publish requires scanner availability
// when configured as mandatory.
//
// Implementation starts in Phase 07.
package security

// Service is a placeholder that will hold security scanning logic starting in Phase 07.
type Service struct{}

