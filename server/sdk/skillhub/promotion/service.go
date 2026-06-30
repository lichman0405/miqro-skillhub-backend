package promotion

import (
	"context"
	"fmt"
	"time"

	"miqro-skillhub/server/sdk/skillhub/eventbus"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/review"
	"miqro-skillhub/server/sdk/skillhub/skill"
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

	updated, err := svc.promotionRequestRepo.UpdateStatusWithVersion(
		ctx, promotionID, string(review.ReviewStatusApproved), reviewerID, comment, nil, req.Version)
	if err != nil {
		return nil, fmt.Errorf("promotion: update status: %w", err)
	}
	if updated == 0 {
		return nil, fmt.Errorf("promotion: concurrent modification")
	}

	// Re-fetch after update.
	approvedReq, err := svc.promotionRequestRepo.FindByID(ctx, promotionID)
	if err != nil || approvedReq == nil {
		return nil, fmt.Errorf("promotion.not_found %d", promotionID)
	}

	sourceSkill, err := svc.skillRepo.FindByID(ctx, approvedReq.SourceSkillID)
	if err != nil {
		return nil, fmt.Errorf("promotion: find source skill: %w", err)
	}
	if sourceSkill == nil {
		return nil, fmt.Errorf("skill.not_found %d", approvedReq.SourceSkillID)
	}

	sourceVersion, err := svc.skillVersionRepo.FindByID(ctx, approvedReq.SourceVersionID)
	if err != nil {
		return nil, fmt.Errorf("promotion: find source version: %w", err)
	}
	if sourceVersion == nil {
		return nil, fmt.Errorf("skill_version.not_found %d", approvedReq.SourceVersionID)
	}

	// Check target skill doesn't already exist for this owner+slug.
	existing, _ := svc.skillRepo.FindByNamespaceIDSlugOwner(ctx, approvedReq.TargetNamespaceID, sourceSkill.Slug, sourceSkill.OwnerID)
	if existing != nil {
		return nil, fmt.Errorf("promotion.target_skill_conflict %s", sourceSkill.Slug)
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
	newSkill, err = svc.skillRepo.Save(ctx, newSkill)
	if err != nil {
		return nil, fmt.Errorf("promotion.target_skill_conflict %s: %w", sourceSkill.Slug, err)
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
	newVersion, err = svc.skillVersionRepo.Save(ctx, newVersion)
	if err != nil {
		return nil, fmt.Errorf("promotion: save new version: %w", err)
	}

	// Update skill's latest version.
	newSkill.LatestVersionID = &newVersion.ID
	if _, err := svc.skillRepo.Save(ctx, newSkill); err != nil {
		return nil, fmt.Errorf("promotion: save new skill: %w", err)
	}

	// Copy file records (reuse storageKey).
	sourceFiles, err := svc.skillFileRepo.FindByVersionID(ctx, approvedReq.SourceVersionID)
	if err != nil {
		return nil, fmt.Errorf("promotion: find source files: %w", err)
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
		if _, err := svc.skillFileRepo.SaveAll(ctx, copiedFiles); err != nil {
			return nil, fmt.Errorf("promotion: save files: %w", err)
		}
	}

	// Update promotion request with target skill id.
	approvedReq.TargetSkillID = &newSkill.ID
	savedReq, err := svc.promotionRequestRepo.Save(ctx, *approvedReq)
	if err != nil {
		return nil, fmt.Errorf("promotion: save request: %w", err)
	}

	svc.publishEvent(ctx, SkillPublishedEvent{
		SkillID:        newSkill.ID,
		SkillVersionID: newVersion.ID,
		PublishedBy:    reviewerID,
	})
	svc.publishEvent(ctx, PromotionApprovedEvent{
		RequestID:     approvedReq.ID,
		SourceSkillID: approvedReq.SourceSkillID,
		ReviewedBy:    reviewerID,
		SubmittedBy:   approvedReq.SubmittedBy,
	})

	if svc.notifier != nil {
		_ = svc.notifier.NotifyUser(ctx, approvedReq.SubmittedBy, "PROMOTION", "PROMOTION_REQUEST", promotionID,
			"Promotion approved", `{"status":"APPROVED"}`)
	}

	return &savedReq, nil
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
