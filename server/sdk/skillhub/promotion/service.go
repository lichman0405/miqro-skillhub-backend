package promotion

import (
	"context"
	"fmt"
	"time"

	"miqro-skillhub/server/sdk/skillhub/eventbus"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/review"
	"miqro-skillhub/server/sdk/skillhub/skill"
	"miqro-skillhub/server/sdk/skillhub/uow"
)

// PromotionNotifier sends governance notifications for promotion events.
type PromotionNotifier interface {
	NotifyUser(ctx context.Context, userID, category, entityType string, entityID int64, title, bodyJSON string) error
}

// PromotionService handles promotion requests that copy approved skills
// into the global namespace.
// Mirrors source com.iflytek.skillhub.domain.review.PromotionService.
type PromotionService struct {
	promotionRequestRepo review.PromotionRequestRepository
	skillRepo            skill.SkillRepository
	skillVersionRepo     skill.SkillVersionRepository
	skillFileRepo        skill.SkillFileRepository
	namespaceRepo        namespace.NamespaceRepository
	permissionChecker    *review.ReviewPermissionChecker
	eventBus             eventbus.Bus
	notifier             PromotionNotifier
	transactor           uow.Transactor
}

// SetTransactor injects an optional transaction boundary.  When non-nil,
// ApprovePromotion wraps all database writes inside a single WithinTx call.
func (svc *PromotionService) SetTransactor(tx uow.Transactor) {
	svc.transactor = tx
}

func (svc *PromotionService) withinTx(ctx context.Context, fn func(context.Context) error) error {
	if svc.transactor == nil {
		return fn(ctx)
	}
	return svc.transactor.WithinTx(ctx, fn)
}

// NewPromotionService creates a PromotionService.
func NewPromotionService(
	promotionRequestRepo review.PromotionRequestRepository,
	skillRepo skill.SkillRepository,
	skillVersionRepo skill.SkillVersionRepository,
	skillFileRepo skill.SkillFileRepository,
	namespaceRepo namespace.NamespaceRepository,
	permissionChecker *review.ReviewPermissionChecker,
	eventBus eventbus.Bus,
	notifier PromotionNotifier,
) *PromotionService {
	if permissionChecker == nil {
		permissionChecker = review.NewReviewPermissionChecker()
	}
	return &PromotionService{
		promotionRequestRepo: promotionRequestRepo,
		skillRepo:            skillRepo,
		skillVersionRepo:     skillVersionRepo,
		skillFileRepo:        skillFileRepo,
		namespaceRepo:        namespaceRepo,
		permissionChecker:    permissionChecker,
		eventBus:             eventBus,
		notifier:             notifier,
	}
}

// SubmitPromotion submits a promotion request for a published source version.
func (svc *PromotionService) SubmitPromotion(
	ctx context.Context,
	sourceSkillID int64,
	sourceVersionID int64,
	targetNamespaceID int64,
	actorID string,
	userNsRoles map[int64]string,
	platformRoles map[string]bool,
) (*review.PromotionRequest, error) {
	sourceSkill, err := svc.skillRepo.FindByID(ctx, sourceSkillID)
	if err != nil {
		return nil, fmt.Errorf("promotion: find skill: %w", err)
	}
	if sourceSkill == nil {
		return nil, fmt.Errorf("skill.not_found %d", sourceSkillID)
	}

	sourceVersion, err := svc.skillVersionRepo.FindByID(ctx, sourceVersionID)
	if err != nil {
		return nil, fmt.Errorf("promotion: find version: %w", err)
	}
	if sourceVersion == nil {
		return nil, fmt.Errorf("skill_version.not_found %d", sourceVersionID)
	}

	if sourceVersion.SkillID != sourceSkillID {
		return nil, fmt.Errorf("promotion.version_skill_mismatch %d %d", sourceVersionID, sourceSkillID)
	}

	if sourceVersion.Status != "PUBLISHED" {
		return nil, fmt.Errorf("promotion.version_not_published %d", sourceVersionID)
	}

	sourceNamespace, err := svc.namespaceRepo.FindByID(ctx, sourceSkill.NamespaceID)
	if err != nil {
		return nil, fmt.Errorf("promotion: find namespace: %w", err)
	}
	if sourceNamespace == nil {
		return nil, fmt.Errorf("namespace.not_found %d", sourceSkill.NamespaceID)
	}
	if err := assertNamespaceActive(sourceNamespace); err != nil {
		return nil, err
	}

	if !svc.permissionChecker.CanSubmitPromotion(sourceSkill.OwnerID, sourceSkill.NamespaceID, actorID, userNsRoles, platformRoles) {
		return nil, fmt.Errorf("promotion.submit.no_permission")
	}

	targetNamespace, err := svc.namespaceRepo.FindByID(ctx, targetNamespaceID)
	if err != nil {
		return nil, fmt.Errorf("promotion: find target namespace: %w", err)
	}
	if targetNamespace == nil {
		return nil, fmt.Errorf("namespace.not_found %d", targetNamespaceID)
	}

	if targetNamespace.Type != "GLOBAL" {
		return nil, fmt.Errorf("promotion.target_not_global %d", targetNamespaceID)
	}

	// Reject duplicate pending promotion.
	existingPending, _ := svc.promotionRequestRepo.FindBySourceSkillIDAndStatus(ctx, sourceSkillID, string(review.ReviewStatusPending))
	if existingPending != nil {
		return nil, fmt.Errorf("promotion.duplicate_pending %d", sourceVersionID)
	}

	// Reject already approved promotion.
	existingApproved, _ := svc.promotionRequestRepo.FindBySourceSkillIDAndStatus(ctx, sourceSkillID, string(review.ReviewStatusApproved))
	if existingApproved != nil {
		return nil, fmt.Errorf("promotion.already_promoted %d", sourceSkillID)
	}

	now := time.Now()
	req := review.PromotionRequest{
		SourceSkillID:     sourceSkillID,
		SourceVersionID:   sourceVersionID,
		TargetNamespaceID: targetNamespaceID,
		SubmittedBy:       actorID,
		Status:            string(review.ReviewStatusPending),
		Version:           1,
		SubmittedAt:       now,
	}
	saved, err := svc.promotionRequestRepo.Save(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("promotion: save: %w", err)
	}

	svc.publishEvent(ctx, PromotionSubmittedEvent{
		RequestID:       saved.ID,
		SourceSkillID:   saved.SourceSkillID,
		SourceVersionID: saved.SourceVersionID,
		SubmittedBy:     saved.SubmittedBy,
	})

	return &saved, nil
}

// ApprovePromotion approves a pending promotion request and materializes a
// published copy of the source version in the target global namespace.
func (svc *PromotionService) ApprovePromotion(
	ctx context.Context,
	promotionID int64,
	reviewerID string,
	comment string,
	platformRoles map[string]bool,
) (*review.PromotionRequest, error) {
	req, err := svc.promotionRequestRepo.FindByID(ctx, promotionID)
	if err != nil {
		return nil, fmt.Errorf("promotion: find: %w", err)
	}
	if req == nil {
		return nil, fmt.Errorf("promotion.not_found %d", promotionID)
	}

	if !review.IsPending(req.Status) {
		return nil, fmt.Errorf("promotion.not_pending %d", promotionID)
	}

	if !svc.permissionChecker.CanReviewPromotion(req.SubmittedBy, reviewerID, platformRoles) {
		return nil, fmt.Errorf("promotion.no_permission")
	}

	// Capture values needed outside the transaction for events/notifications.
	var savedReq *review.PromotionRequest
	var publishedSkillID int64
	var publishedVersionID int64
	var submittedBy string
	var sourceSkillBefore int64

	err = svc.withinTx(ctx, func(txCtx context.Context) error {
		updated, uErr := svc.promotionRequestRepo.UpdateStatusWithVersion(
			txCtx, promotionID, string(review.ReviewStatusApproved), reviewerID, comment, nil, req.Version)
		if uErr != nil {
			return fmt.Errorf("promotion: update status: %w", uErr)
		}
		if updated == 0 {
			return fmt.Errorf("promotion: concurrent modification")
		}

		// Re-fetch after update.
		approvedReq, uErr := svc.promotionRequestRepo.FindByID(txCtx, promotionID)
		if uErr != nil || approvedReq == nil {
			return fmt.Errorf("promotion.not_found %d", promotionID)
		}

		sourceSkill, uErr := svc.skillRepo.FindByID(txCtx, approvedReq.SourceSkillID)
		if uErr != nil {
			return fmt.Errorf("promotion: find source skill: %w", uErr)
		}
		if sourceSkill == nil {
			return fmt.Errorf("skill.not_found %d", approvedReq.SourceSkillID)
		}

		sourceVersion, uErr := svc.skillVersionRepo.FindByID(txCtx, approvedReq.SourceVersionID)
		if uErr != nil {
			return fmt.Errorf("promotion: find source version: %w", uErr)
		}
		if sourceVersion == nil {
			return fmt.Errorf("skill_version.not_found %d", approvedReq.SourceVersionID)
		}

		// Check target skill doesn't already exist for this owner+slug.
		existing, _ := svc.skillRepo.FindByNamespaceIDSlugOwner(txCtx, approvedReq.TargetNamespaceID, sourceSkill.Slug, sourceSkill.OwnerID)
		if existing != nil {
			return fmt.Errorf("promotion.target_skill_conflict %s", sourceSkill.Slug)
		}

		// Create new skill in global namespace.
		now := time.Now()
		newSkill := skill.Skill{
			NamespaceID:      approvedReq.TargetNamespaceID,
			Slug:             sourceSkill.Slug,
			DisplayName:      sourceSkill.DisplayName,
			Summary:          sourceSkill.Summary,
			OwnerID:          sourceSkill.OwnerID,
			SourceSkillID:    &sourceSkill.ID,
			Visibility:       "PUBLIC",
			Status:           "ACTIVE",
			CreatedBy:        &reviewerID,
			UpdatedBy:        &reviewerID,
			LatestVersionID:  nil, // set after version creation
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		newSkill, uErr = svc.skillRepo.Save(txCtx, newSkill)
		if uErr != nil {
			return fmt.Errorf("promotion.target_skill_conflict %s: %w", sourceSkill.Slug, uErr)
		}

		// Create new version copying metadata from source.
		newVersion := skill.SkillVersion{
			SkillID:             newSkill.ID,
			Version:             sourceVersion.Version,
			Status:              "PUBLISHED",
			PublishedAt:         &now,
			RequestedVisibility: strPtr("PUBLIC"),
			Changelog:           sourceVersion.Changelog,
			ParsedMetadataJSON:  sourceVersion.ParsedMetadataJSON,
			ManifestJSON:        sourceVersion.ManifestJSON,
			FileCount:           sourceVersion.FileCount,
			TotalSize:           sourceVersion.TotalSize,
			BundleReady:         sourceVersion.BundleReady,
			DownloadReady:       sourceVersion.DownloadReady,
			CreatedBy:           sourceVersion.CreatedBy,
			CreatedAt:           now,
		}
		newVersion, uErr = svc.skillVersionRepo.Save(txCtx, newVersion)
		if uErr != nil {
			return fmt.Errorf("promotion: save new version: %w", uErr)
		}

		// Update skill's latest version.
		newSkill.LatestVersionID = &newVersion.ID
		if _, uErr := svc.skillRepo.Save(txCtx, newSkill); uErr != nil {
			return fmt.Errorf("promotion: save new skill: %w", uErr)
		}

		// Copy file records (reuse storageKey).
		sourceFiles, uErr := svc.skillFileRepo.FindByVersionID(txCtx, approvedReq.SourceVersionID)
		if uErr != nil {
			return fmt.Errorf("promotion: find source files: %w", uErr)
		}
		copiedFiles := make([]skill.SkillFile, 0, len(sourceFiles))
		for _, f := range sourceFiles {
			copiedFiles = append(copiedFiles, skill.SkillFile{
				VersionID:   newVersion.ID,
				FilePath:    f.FilePath,
				FileSize:    f.FileSize,
				ContentType: f.ContentType,
				SHA256:      f.SHA256,
				StorageKey:  f.StorageKey,
				CreatedAt:   now,
			})
		}
		if len(copiedFiles) > 0 {
			if _, uErr := svc.skillFileRepo.SaveAll(txCtx, copiedFiles); uErr != nil {
				return fmt.Errorf("promotion: save files: %w", uErr)
			}
		}

		// Update promotion request with target skill id.
		approvedReq.TargetSkillID = &newSkill.ID
		reqToReturn, uErr := svc.promotionRequestRepo.Save(txCtx, *approvedReq)
		if uErr != nil {
			return fmt.Errorf("promotion: save request: %w", uErr)
		}

		savedReq = &reqToReturn
		publishedSkillID = newSkill.ID
		publishedVersionID = newVersion.ID
		submittedBy = approvedReq.SubmittedBy
		sourceSkillBefore = approvedReq.SourceSkillID
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Emit events and notifications only after the transaction commits.
	svc.publishEvent(ctx, SkillPublishedEvent{
		SkillID:        publishedSkillID,
		SkillVersionID: publishedVersionID,
		PublishedBy:    reviewerID,
	})
	svc.publishEvent(ctx, PromotionApprovedEvent{
		RequestID:     savedReq.ID,
		SourceSkillID: sourceSkillBefore,
		ReviewedBy:    reviewerID,
		SubmittedBy:   submittedBy,
	})

	if svc.notifier != nil {
		_ = svc.notifier.NotifyUser(ctx, submittedBy, "PROMOTION", "PROMOTION_REQUEST", promotionID,
			"Promotion approved", `{"status":"APPROVED"}`)
	}

	return savedReq, nil
}

// RejectPromotion rejects a pending promotion request without changing the source skill.
func (svc *PromotionService) RejectPromotion(
	ctx context.Context,
	promotionID int64,
	reviewerID string,
	comment string,
	platformRoles map[string]bool,
) (*review.PromotionRequest, error) {
	req, err := svc.promotionRequestRepo.FindByID(ctx, promotionID)
	if err != nil {
		return nil, fmt.Errorf("promotion: find: %w", err)
	}
	if req == nil {
		return nil, fmt.Errorf("promotion.not_found %d", promotionID)
	}

	if !review.IsPending(req.Status) {
		return nil, fmt.Errorf("promotion.not_pending %d", promotionID)
	}

	if !svc.permissionChecker.CanReviewPromotion(req.SubmittedBy, reviewerID, platformRoles) {
		return nil, fmt.Errorf("promotion.no_permission")
	}

	commentPtr := &comment
	if comment == "" {
		commentPtr = nil
	}
	updated, err := svc.promotionRequestRepo.UpdateStatusWithVersion(
		ctx, promotionID, string(review.ReviewStatusRejected), reviewerID, comment, nil, req.Version)
	if err != nil {
		return nil, fmt.Errorf("promotion: update status: %w", err)
	}
	if updated == 0 {
		return nil, fmt.Errorf("promotion: concurrent modification")
	}

	req.Status = string(review.ReviewStatusRejected)
	req.ReviewedBy = &reviewerID
	req.ReviewComment = commentPtr
	now := time.Now()
	req.ReviewedAt = &now

	svc.publishEvent(ctx, PromotionRejectedEvent{
		RequestID:     req.ID,
		SourceSkillID: req.SourceSkillID,
		ReviewedBy:    reviewerID,
		SubmittedBy:   req.SubmittedBy,
		Comment:       comment,
	})

	if svc.notifier != nil {
		_ = svc.notifier.NotifyUser(ctx, req.SubmittedBy, "PROMOTION", "PROMOTION_REQUEST", promotionID,
			"Promotion rejected", `{"status":"REJECTED"}`)
	}

	return req, nil
}

// WithdrawPromotion withdraws a pending promotion request.  Only the submitter
// or SUPER_ADMIN may withdraw.  The request is deleted from the repository.
func (svc *PromotionService) WithdrawPromotion(
	ctx context.Context,
	promotionID int64,
	actorID string,
	platformRoles map[string]bool,
) error {
	req, err := svc.promotionRequestRepo.FindByID(ctx, promotionID)
	if err != nil {
		return fmt.Errorf("promotion: find: %w", err)
	}
	if req == nil {
		return fmt.Errorf("promotion.not_found %d", promotionID)
	}

	if !review.IsPending(req.Status) {
		return fmt.Errorf("promotion.not_pending %d", promotionID)
	}

	// Only submitter may withdraw, unless SUPER_ADMIN.
	if req.SubmittedBy != actorID && !platformRoles["SUPER_ADMIN"] {
		return fmt.Errorf("promotion.withdraw.not_submitter")
	}

	return svc.promotionRequestRepo.Delete(ctx, promotionID)
}

func (svc *PromotionService) publishEvent(ctx context.Context, event eventbus.Event) {
	if svc.eventBus != nil {
		_ = svc.eventBus.Publish(ctx, event)
	}
}

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

func strPtr(s string) *string { return &s }
