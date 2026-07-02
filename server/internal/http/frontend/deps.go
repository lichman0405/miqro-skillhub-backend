package frontend

import (
	"context"

	"miqro-skillhub/server/sdk/skillhub/governance"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/review"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// ReviewFrontendDeps carries the narrow set of SDK repositories used by the
// review queue/detail read-model handlers. Handlers must remain projection-only
// and must not perform review mutations.
type ReviewFrontendDeps struct {
	ReviewTasks review.ReviewTaskRepository
	Versions    skill.SkillVersionRepository
	Skills      skill.SkillRepository
	Namespaces  namespace.NamespaceRepository
}

// PromotionFrontendDeps carries the narrow set of SDK repositories used by the
// promotion queue/detail read-model handlers.
type PromotionFrontendDeps struct {
	PromotionRequests review.PromotionRequestRepository
	Versions          skill.SkillVersionRepository
	Skills            skill.SkillRepository
	Namespaces        namespace.NamespaceRepository
}

// GovernanceFrontendDeps carries the notification service and optional count
// repositories used by the governance workbench read-model handler.
type GovernanceFrontendDeps struct {
	Notifications     *governance.GovernanceNotificationService
	ReviewTasks       review.ReviewTaskRepository
	PromotionRequests review.PromotionRequestRepository
}

// AdminStatsQuery abstracts aggregate stats retrieval for the admin dashboard.
// It is implemented by the postgres adapter so the frontend handler stays free
// of SQL.
type AdminStatsQuery interface {
	Stats(ctx context.Context) (AdminStatsView, error)
}

// AdminFrontendDeps carries the query interface used by the admin dashboard
// read-model handler.
type AdminFrontendDeps struct {
	Stats AdminStatsQuery
}
