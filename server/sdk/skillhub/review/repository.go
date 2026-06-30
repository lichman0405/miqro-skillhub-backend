package review

import "context"

// ReviewTaskRepository defines the persistence contract for review tasks.
type ReviewTaskRepository interface {
	Save(ctx context.Context, task ReviewTask) (ReviewTask, error)
	FindByID(ctx context.Context, id int64) (*ReviewTask, error)
	FindByVersionIDAndStatus(ctx context.Context, versionID int64, status string) (*ReviewTask, error)
	FindByStatus(ctx context.Context, status string) ([]ReviewTask, error)
	FindByNamespaceIDAndStatus(ctx context.Context, namespaceID int64, status string) ([]ReviewTask, error)
	FindBySubmittedByAndStatus(ctx context.Context, submittedBy string, status string) ([]ReviewTask, error)
	ExistsByNamespaceID(ctx context.Context, namespaceID int64) (bool, error)
	Delete(ctx context.Context, id int64) error
	DeleteByVersionIDs(ctx context.Context, versionIDs []int64) error
	UpdateStatusWithVersion(ctx context.Context, id int64, status string, reviewedBy string, reviewComment string, expectedVersion int) (int, error)
}

// PromotionRequestRepository defines the persistence contract for promotion requests.
type PromotionRequestRepository interface {
	Save(ctx context.Context, req PromotionRequest) (PromotionRequest, error)
	FindByID(ctx context.Context, id int64) (*PromotionRequest, error)
	FindBySourceVersionIDAndStatus(ctx context.Context, versionID int64, status string) (*PromotionRequest, error)
	FindBySourceSkillIDAndStatus(ctx context.Context, skillID int64, status string) (*PromotionRequest, error)
	FindByStatus(ctx context.Context, status string) ([]PromotionRequest, error)
	ExistsByTargetNamespaceID(ctx context.Context, namespaceID int64) (bool, error)
	DeleteBySourceOrTargetSkillID(ctx context.Context, skillID int64) error
	UpdateStatusWithVersion(ctx context.Context, id int64, status string, reviewedBy string, reviewComment string, targetSkillID *int64, expectedVersion int) (int, error)
}
