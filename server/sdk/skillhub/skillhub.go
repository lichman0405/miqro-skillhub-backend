// Package skillhub is the public Go SDK for SkillHub behavior.
//
// It exposes service facades that embeddable programs, HTTP adapters, CLI
// tools, and tests call for core domain operations.  All public SDK packages
// live under server/sdk/skillhub; concrete adapters (PostgreSQL, Redis, S3,
// local filesystem, HTTP) live under server/internal/adapters and
// server/internal/http.
//
// The root Service struct holds typed fields for every domain service so
// callers can compose and inject them.  Fields are nil in early phases and
// filled as implementation progresses.
package skillhub

import (
	"miqro-skillhub/server/sdk/skillhub/auth"
	"miqro-skillhub/server/sdk/skillhub/audit"
	"miqro-skillhub/server/sdk/skillhub/governance"
	"miqro-skillhub/server/sdk/skillhub/label"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/notification"
	"miqro-skillhub/server/sdk/skillhub/promotion"
	"miqro-skillhub/server/sdk/skillhub/report"
	"miqro-skillhub/server/sdk/skillhub/review"
	"miqro-skillhub/server/sdk/skillhub/search"
	"miqro-skillhub/server/sdk/skillhub/security"
	"miqro-skillhub/server/sdk/skillhub/skill"
	"miqro-skillhub/server/sdk/skillhub/social"
)

// Service is the root facade for all SkillHub domain services.
//
// Inject the concrete service implementations your process needs; leave a
// field nil when the corresponding domain is not yet implemented.
type Service struct {
	Auth          *auth.Service
	Namespaces    *namespace.Service
	Skills        *skill.Service
	Reviews       *review.ReviewService
	Promotions    *promotion.PromotionService
	Labels        *label.Service
	Search        *search.Service
	Social        *social.Service
	Reports       *report.Service
	Governance    *governance.GovernanceNotificationService
	Notifications *notification.NotificationService
	Security      *security.Service
	Audit         *audit.AuditLogService
}
