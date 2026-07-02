package review

import (
	"context"
	"fmt"
	"time"

	"miqro-skillhub/server/sdk/skillhub/eventbus"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// GateEnforcer is called before review approval to check CI gate policies.
// If gates are not satisfied, the approval is blocked. Returns nil if gates pass.
type GateEnforcer func(ctx context.Context, skillID, versionID int64, triggerType string) error

// ReviewNotifier sends governance notifications for review events.
type ReviewNotifier interface {
	NotifyUser(ctx context.Context, userID, category, entityType string, entityID int64, title, bodyJSON string) error
}

// ReviewService coordinates the review workflow for skill versions.
// Mirrors source com.iflytek.skillhub.domain.review.ReviewService.
type ReviewService struct {
	reviewTaskRepo  ReviewTaskRepository
	skillVersionRepo skill.SkillVersionRepository
	skillRepo       skill.SkillRepository
	namespaceRepo   namespace.NamespaceRepository
	permissionChecker *ReviewPermissionChecker
	eventBus        eventbus.Bus
	notifier        ReviewNotifier
	gateEnforcer    GateEnforcer // optional: CI gate enforcement before approval
}

// SetGateEnforcer injects a gate enforcement function for CI gates.
// When set, ApproveReview will call it before publishing the version.
func (svc *ReviewService) SetGateEnforcer(enforcer GateEnforcer) {
	svc.gateEnforcer = enforcer
}

// NewReviewService creates a ReviewService.
func NewReviewService(
	reviewTaskRepo ReviewTaskRepository,
	skillVersionRepo skill.SkillVersionRepository,
	skillRepo skill.SkillRepository,
	namespaceRepo namespace.NamespaceRepository,
	permissionChecker *ReviewPermissionChecker,
	eventBus eventbus.Bus,
	notifier ReviewNotifier,
) *ReviewService {
	if permissionChecker == nil {
		permissionChecker = NewReviewPermissionChecker()
	}
	return &ReviewService{
		reviewTaskRepo:   reviewTaskRepo,
		skillVersionRepo: skillVersionRepo,
		skillRepo:        skillRepo,
		namespaceRepo:    namespaceRepo,
		permissionChecker: permissionChecker,
		eventBus:         eventBus,
		notifier:         notifier,
	}
}

// SubmitReview submits a version into the review queue.
// Accepts both DRAFT (legacy) and UPLOADED versions.
func (svc *ReviewService) SubmitReview(
	ctx context.Context,
	skillVersionID int64,
	actorID string,
	userNsRoles map[int64]string,
	platformRoles map[string]bool,
) (*ReviewTask, error) {
	skillVersion, err := svc.skillVersionRepo.FindByID(ctx, skillVersionID)
	if err != nil {
		return nil, fmt.Errorf("review: find version: %w", err)
	}
	if skillVersion == nil {
		return nil, fmt.Errorf("skill_version.not_found %d", skillVersionID)
	}

	sk, err := svc.skillRepo.FindByID(ctx, skillVersion.SkillID)
	if err != nil {
		return nil, fmt.Errorf("review: find skill: %w", err)
	}
	if sk == nil {
		return nil, fmt.Errorf("skill.not_found %d", skillVersion.SkillID)
	}

	ns, err := svc.namespaceRepo.FindByID(ctx, sk.NamespaceID)
	if err != nil {
		return nil, fmt.Errorf("review: find namespace: %w", err)
	}
	if ns == nil {
		return nil, fmt.Errorf("namespace.not_found %d", sk.NamespaceID)
	}
	if err := assertNamespaceActive(ns); err != nil {
		return nil, err
	}

	if !svc.permissionChecker.CanSubmitForReview(sk.OwnerID, sk.NamespaceID, actorID, userNsRoles, platformRoles) {
		return nil, fmt.Errorf("review.submit.no_permission")
	}

	// Support both DRAFT (legacy) and UPLOADED.
	if skillVersion.Status != "DRAFT" && skillVersion.Status != "UPLOADED" {
		return nil, fmt.Errorf("review.submit.not_draft %d", skillVersionID)
	}

	skillVersion.Status = "PENDING_REVIEW"
	if _, err := svc.skillVersionRepo.Save(ctx, *skillVersion); err != nil {
		return nil, fmt.Errorf("review: save version: %w", err)
	}

	task := ReviewTask{
		SkillVersionID: skillVersionID,
		NamespaceID:    sk.NamespaceID,
		SubmittedBy:    actorID,
		Status:         string(ReviewStatusPending),
		Version:        1,
		SubmittedAt:    time.Now(),
	}
	saved, err := svc.reviewTaskRepo.Save(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("review.submit.duplicate %d: %w", skillVersionID, err)
	}

	svc.publishEvent(ctx, ReviewSubmittedEvent{
		TaskID:         saved.ID,
		SkillID:        skillVersion.SkillID,
		SkillVersionID: skillVersion.ID,
		SubmittedBy:    saved.SubmittedBy,
		NamespaceID:    saved.NamespaceID,
	})

	return &saved, nil
}

// ApproveReview approves a pending review task, publishes the version, and
// updates the skill's latest version pointer.
func (svc *ReviewService) ApproveReview(
	ctx context.Context,
	reviewTaskID int64,
	reviewerID string,
	comment string,
	userNsRoles map[int64]string,
	platformRoles map[string]bool,
) (*ReviewTask, error) {
	task, err := svc.reviewTaskRepo.FindByID(ctx, reviewTaskID)
	if err != nil {
		return nil, fmt.Errorf("review: find task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("review_task.not_found %d", reviewTaskID)
	}

	if !IsPending(task.Status) {
		return nil, fmt.Errorf("review.not_pending %d", reviewTaskID)
	}

	ns, err := svc.namespaceRepo.FindByID(ctx, task.NamespaceID)
	if err != nil {
		return nil, fmt.Errorf("review: find namespace: %w", err)
	}
	if ns == nil {
		return nil, fmt.Errorf("namespace.not_found %d", task.NamespaceID)
	}
	if err := assertNamespaceActive(ns); err != nil {
		return nil, err
	}
	nsType := ns.Type
	if nsType == "" {
		nsType = "TEAM"
	}

	if !svc.permissionChecker.CanReview(task.SubmittedBy, task.NamespaceID, nsType, reviewerID, userNsRoles, platformRoles) {
		return nil, fmt.Errorf("review.no_permission")
	}

	skillVersion, err := svc.skillVersionRepo.FindByID(ctx, task.SkillVersionID)
	if err != nil {
		return nil, fmt.Errorf("review: find version: %w", err)
	}
	if skillVersion == nil {
		return nil, fmt.Errorf("skill_version.not_found %d", task.SkillVersionID)
	}
	if skillVersion.Status == "SCANNING" {
		return nil, fmt.Errorf("review.approve.scan_in_progress %d", reviewTaskID)
	}

	commentPtr := &comment
	if comment == "" {
		commentPtr = nil
	}
	updated, err := svc.reviewTaskRepo.UpdateStatusWithVersion(
		ctx, reviewTaskID, string(ReviewStatusApproved), reviewerID, comment, task.Version)
	if err != nil {
		return nil, fmt.Errorf("review: update status: %w", err)
	}
	if updated == 0 {
		return nil, fmt.Errorf("review: concurrent modification")
	}
	// In-memory sync after optimistic update.
	task.Status = string(ReviewStatusApproved)
	task.ReviewedBy = &reviewerID
	task.ReviewComment = commentPtr
	now := time.Now()
	task.ReviewedAt = &now

	sk, err := svc.skillRepo.FindByID(ctx, skillVersion.SkillID)
	if err != nil {
		return nil, fmt.Errorf("review: find skill: %w", err)
	}
	if sk == nil {
		return nil, fmt.Errorf("skill.not_found %d", skillVersion.SkillID)
	}

	// Check no other owner has a published skill with the same slug.
	sameSlugSkills, err := svc.skillRepo.FindByNamespaceIDAndSlug(ctx, sk.NamespaceID, sk.Slug)
	if err != nil {
		return nil, fmt.Errorf("review: find same-slug: %w", err)
	}
	for _, other := range sameSlugSkills {
		if other.ID == sk.ID {
			continue
		}
		publishedVersions, _ := svc.skillVersionRepo.FindBySkillIDAndStatus(ctx, other.ID, "PUBLISHED")
		if len(publishedVersions) > 0 {
			return nil, fmt.Errorf("error.skill.approve.nameConflict %s", sk.Slug)
		}
	}

	// ── Gate enforcement ──────────────────────────────────────────────────
	// Before publishing the version via review approval, check that CI gates
	// are satisfied. This is SDK-level enforcement, not just HTTP handler level.
	if svc.gateEnforcer != nil {
		if err := svc.gateEnforcer(ctx, sk.ID, skillVersion.ID, "review_approve"); err != nil {
			return nil, fmt.Errorf("review: gate enforcement: %w", err)
		}
	}
	// ── End gate enforcement ──────────────────────────────────────────────

	// Publish the version.
	skillVersion.Status = "PUBLISHED"
	skillVersion.PublishedAt = &now
	if _, err := svc.skillVersionRepo.Save(ctx, *skillVersion); err != nil {
		return nil, fmt.Errorf("review: save version: %w", err)
	}

	// Update skill latest version pointer and visibility.
	sk.LatestVersionID = &skillVersion.ID
	if skillVersion.RequestedVisibility != nil {
		sk.Visibility = *skillVersion.RequestedVisibility
	}
	if skillVersion.ParsedMetadataJSON != nil {
		sk.DisplayName = skillVersion.Version // fallback; real metadata deserialization deferred
	}
	sk.UpdatedBy = &reviewerID
	if _, err := svc.skillRepo.Save(ctx, *sk); err != nil {
		return nil, fmt.Errorf("review: save skill: %w", err)
	}

	svc.publishEvent(ctx, SkillPublishedEvent{
		SkillID:        sk.ID,
		SkillVersionID: skillVersion.ID,
		PublishedBy:    reviewerID,
	})
	svc.publishEvent(ctx, ReviewApprovedEvent{
		TaskID:         task.ID,
		SkillID:        sk.ID,
		SkillVersionID: skillVersion.ID,
		ReviewedBy:     reviewerID,
		SubmittedBy:    task.SubmittedBy,
	})

	if svc.notifier != nil {
		_ = svc.notifier.NotifyUser(ctx, task.SubmittedBy, "REVIEW", "REVIEW_TASK", task.ID,
			"Review approved", `{"status":"APPROVED"}`)
	}

	return task, nil
}

// RejectReview rejects a pending review task and returns the version to a
// non-published state.
func (svc *ReviewService) RejectReview(
	ctx context.Context,
	reviewTaskID int64,
	reviewerID string,
	comment string,
	userNsRoles map[int64]string,
	platformRoles map[string]bool,
) (*ReviewTask, error) {
	task, err := svc.reviewTaskRepo.FindByID(ctx, reviewTaskID)
	if err != nil {
		return nil, fmt.Errorf("review: find task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("review_task.not_found %d", reviewTaskID)
	}

	if !IsPending(task.Status) {
		return nil, fmt.Errorf("review.not_pending %d", reviewTaskID)
	}

	ns, err := svc.namespaceRepo.FindByID(ctx, task.NamespaceID)
	if err != nil {
		return nil, fmt.Errorf("review: find namespace: %w", err)
	}
	if ns == nil {
		return nil, fmt.Errorf("namespace.not_found %d", task.NamespaceID)
	}
	if err := assertNamespaceActive(ns); err != nil {
		return nil, err
	}
	nsType := ns.Type
	if nsType == "" {
		nsType = "TEAM"
	}

	if !svc.permissionChecker.CanReview(task.SubmittedBy, task.NamespaceID, nsType, reviewerID, userNsRoles, platformRoles) {
		return nil, fmt.Errorf("review.no_permission")
	}

	commentPtr := &comment
	if comment == "" {
		commentPtr = nil
	}
	updated, err := svc.reviewTaskRepo.UpdateStatusWithVersion(
		ctx, reviewTaskID, string(ReviewStatusRejected), reviewerID, comment, task.Version)
	if err != nil {
		return nil, fmt.Errorf("review: update status: %w", err)
	}
	if updated == 0 {
		return nil, fmt.Errorf("review: concurrent modification")
	}
	task.Status = string(ReviewStatusRejected)
	task.ReviewedBy = &reviewerID
	task.ReviewComment = commentPtr
	now := time.Now()
	task.ReviewedAt = &now

	skillVersion, err := svc.skillVersionRepo.FindByID(ctx, task.SkillVersionID)
	if err != nil {
		return nil, fmt.Errorf("review: find version: %w", err)
	}
	if skillVersion != nil {
		skillVersion.Status = "REJECTED"
		if _, err := svc.skillVersionRepo.Save(ctx, *skillVersion); err != nil {
			return nil, fmt.Errorf("review: save version: %w", err)
		}
	}

	svc.publishEvent(ctx, ReviewRejectedEvent{
		TaskID:         task.ID,
		SkillID:        skillVersion.SkillID,
		SkillVersionID: task.SkillVersionID,
		ReviewedBy:     reviewerID,
		SubmittedBy:    task.SubmittedBy,
		Comment:        comment,
	})

	if svc.notifier != nil {
		_ = svc.notifier.NotifyUser(ctx, task.SubmittedBy, "REVIEW", "REVIEW_TASK", task.ID,
			"Review rejected", `{"status":"REJECTED"}`)
	}

	return task, nil
}

// WithdrawReview withdraws a previously submitted review request and puts the
// version back to UPLOADED so the owner can amend and resubmit.
func (svc *ReviewService) WithdrawReview(
	ctx context.Context,
	skillVersionID int64,
	actorID string,
) (*skill.SkillVersion, error) {
	task, err := svc.reviewTaskRepo.FindByVersionIDAndStatus(ctx, skillVersionID, string(ReviewStatusPending))
	if err != nil {
		return nil, fmt.Errorf("review: find task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("review_task.not_found_for_version %d", skillVersionID)
	}

	if task.SubmittedBy != actorID {
		return nil, fmt.Errorf("review.withdraw.not_submitter")
	}

	if err := svc.reviewTaskRepo.Delete(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("review: delete task: %w", err)
	}

	skillVersion, err := svc.skillVersionRepo.FindByID(ctx, skillVersionID)
	if err != nil {
		return nil, fmt.Errorf("review: find version: %w", err)
	}
	if skillVersion == nil {
		return nil, fmt.Errorf("skill_version.not_found %d", skillVersionID)
	}

	sk, err := svc.skillRepo.FindByID(ctx, skillVersion.SkillID)
	if err != nil {
		return nil, fmt.Errorf("review: find skill: %w", err)
	}
	if sk == nil {
		return nil, fmt.Errorf("skill.not_found %d", skillVersion.SkillID)
	}

	ns, err := svc.namespaceRepo.FindByID(ctx, sk.NamespaceID)
	if err != nil {
		return nil, fmt.Errorf("review: find namespace: %w", err)
	}
	if err := assertNamespaceActive(ns); err != nil {
		return nil, err
	}

	// withdrawPendingVersion: PENDING_REVIEW → UPLOADED.
	if skillVersion.Status != "PENDING_REVIEW" {
		return nil, fmt.Errorf("review.withdraw.not_pending %d", skillVersion.ID)
	}
	skillVersion.Status = "UPLOADED"
	savedVersion, err := svc.skillVersionRepo.Save(ctx, *skillVersion)
	if err != nil {
		return nil, fmt.Errorf("review: save version: %w", err)
	}

	sk.UpdatedBy = &actorID
	if _, err := svc.skillRepo.Save(ctx, *sk); err != nil {
		return nil, fmt.Errorf("review: save skill: %w", err)
	}

	return &savedVersion, nil
}

func (svc *ReviewService) publishEvent(ctx context.Context, event eventbus.Event) {
	if svc.eventBus != nil {
		_ = svc.eventBus.Publish(ctx, event)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func assertNamespaceActive(ns *namespace.Namespace) error {
	if ns == nil {
		return nil
	}
	if ns.Status == "FROZEN" {
		return fmt.Errorf("error.namespace.frozen %s", ns.Slug)
	}
	if ns.Status == "ARCHIVED" {
		return fmt.Errorf("error.namespace.archived %s", ns.Slug)
	}
	return nil
}

// SkillPublishedEvent is emitted when a skill version is published (via review approval).
type SkillPublishedEvent struct {
	SkillID        int64
	SkillVersionID int64
	PublishedBy    string
}

func (e SkillPublishedEvent) EventName() string { return "skill.published" }
