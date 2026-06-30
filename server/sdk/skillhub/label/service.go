package label

import (
	"context"
	"fmt"
	"time"
)

// Service is the public facade for label operations.
type Service struct {
	Definitions  *LabelDefinitionService
	SkillLabels  *SkillLabelService
}

// LabelDefinitionService manages label definitions and translations.
// Mirrors source com.iflytek.skillhub.domain.label.LabelDefinitionService.
type LabelDefinitionService struct {
	defRepo        LabelDefinitionRepository
	translationRepo LabelTranslationRepository
	permChecker    *LabelPermissionChecker
}

// NewLabelDefinitionService creates a LabelDefinitionService.
func NewLabelDefinitionService(
	defRepo LabelDefinitionRepository,
	translationRepo LabelTranslationRepository,
	permChecker *LabelPermissionChecker,
) *LabelDefinitionService {
	if permChecker == nil {
		permChecker = NewLabelPermissionChecker()
	}
	return &LabelDefinitionService{
		defRepo:         defRepo,
		translationRepo: translationRepo,
		permChecker:     permChecker,
	}
}

// Create creates a new label definition. Requires LABEL_ADMIN or SUPER_ADMIN.
func (svc *LabelDefinitionService) Create(
	ctx context.Context,
	slug, labelType string,
	visibleInFilter bool,
	sortOrder int,
	createdBy string,
	platformRoles map[string]bool,
	translations []LabelTranslation,
) (*LabelDefinition, error) {
	if !svc.permChecker.CanManageLabels(platformRoles) {
		return nil, fmt.Errorf("error.label.noPermission")
	}
	if labelType != TypeRecommended && labelType != TypePrivileged {
		return nil, fmt.Errorf("error.label.type.invalid %s", labelType)
	}
	if err := ValidateSlug(slug); err != nil {
		return nil, err
	}

	existing, _ := svc.defRepo.FindBySlug(ctx, slug)
	if existing != nil {
		return nil, fmt.Errorf("error.label.slug.duplicate %s", slug)
	}

	now := time.Now()
	def := LabelDefinition{
		Slug:            slug,
		Type:            labelType,
		VisibleInFilter: visibleInFilter,
		SortOrder:       sortOrder,
		CreatedBy:       &createdBy,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	saved, err := svc.defRepo.Save(ctx, def)
	if err != nil {
		return nil, fmt.Errorf("label: create: %w", err)
	}

	// Save translations.
	if len(translations) > 0 {
		for i := range translations {
			translations[i].LabelID = saved.ID
			translations[i].CreatedAt = now
			translations[i].UpdatedAt = now
		}
		if _, err := svc.translationRepo.SaveAll(ctx, translations); err != nil {
			return nil, fmt.Errorf("label: save translations: %w", err)
		}
	}

	return &saved, nil
}

// Update updates a label definition. Requires LABEL_ADMIN or SUPER_ADMIN.
func (svc *LabelDefinitionService) Update(
	ctx context.Context,
	id int64,
	slug, labelType string,
	visibleInFilter bool,
	sortOrder int,
	platformRoles map[string]bool,
) (*LabelDefinition, error) {
	if !svc.permChecker.CanManageLabels(platformRoles) {
		return nil, fmt.Errorf("error.label.noPermission")
	}

	def, err := svc.defRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("label: find: %w", err)
	}
	if def == nil {
		return nil, fmt.Errorf("error.label.notFound %d", id)
	}

	if slug != "" && slug != def.Slug {
		if err := ValidateSlug(slug); err != nil {
			return nil, err
		}
		existing, _ := svc.defRepo.FindBySlug(ctx, slug)
		if existing != nil && existing.ID != id {
			return nil, fmt.Errorf("error.label.slug.duplicate %s", slug)
		}
		def.Slug = slug
	}
	if labelType != "" {
		if labelType != TypeRecommended && labelType != TypePrivileged {
			return nil, fmt.Errorf("error.label.type.invalid %s", labelType)
		}
		def.Type = labelType
	}
	def.VisibleInFilter = visibleInFilter
	def.SortOrder = sortOrder
	def.UpdatedAt = time.Now()

	saved, err := svc.defRepo.Save(ctx, *def)
	if err != nil {
		return nil, fmt.Errorf("label: update: %w", err)
	}
	return &saved, nil
}

// Delete deletes a label definition and its translations. Requires LABEL_ADMIN or SUPER_ADMIN.
func (svc *LabelDefinitionService) Delete(ctx context.Context, id int64, platformRoles map[string]bool) error {
	if !svc.permChecker.CanManageLabels(platformRoles) {
		return fmt.Errorf("error.label.noPermission")
	}

	if _, err := svc.defRepo.FindByID(ctx, id); err != nil {
		return fmt.Errorf("label: find: %w", err)
	}

	if err := svc.translationRepo.DeleteByLabelID(ctx, id); err != nil {
		return fmt.Errorf("label: delete translations: %w", err)
	}
	if err := svc.defRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("label: delete: %w", err)
	}
	return nil
}

// GetTranslations returns translations for a label.
func (svc *LabelDefinitionService) GetTranslations(ctx context.Context, labelID int64) ([]LabelTranslation, error) {
	return svc.translationRepo.FindByLabelID(ctx, labelID)
}

// SetTranslations replaces translations for a label. Requires LABEL_ADMIN or SUPER_ADMIN.
func (svc *LabelDefinitionService) SetTranslations(
	ctx context.Context,
	labelID int64,
	translations []LabelTranslation,
	platformRoles map[string]bool,
) ([]LabelTranslation, error) {
	if !svc.permChecker.CanManageLabels(platformRoles) {
		return nil, fmt.Errorf("error.label.noPermission")
	}

	// Delete existing.
	if err := svc.translationRepo.DeleteByLabelID(ctx, labelID); err != nil {
		return nil, fmt.Errorf("label: delete old translations: %w", err)
	}

	now := time.Now()
	for i := range translations {
		translations[i].LabelID = labelID
		translations[i].CreatedAt = now
		translations[i].UpdatedAt = now
	}
	return svc.translationRepo.SaveAll(ctx, translations)
}

// ---------------------------------------------------------------------------
// SkillLabelService
// ---------------------------------------------------------------------------

// SkillLabelSearchSyncer is notified when skill labels change so the search index
// can be updated.
type SkillLabelSearchSyncer interface {
	SyncSkillLabels(ctx context.Context, skillID int64) error
}

// SkillLabelService manages label assignments on skills.
// Mirrors source com.iflytek.skillhub.domain.label.SkillLabelService.
type SkillLabelService struct {
	skillLabelRepo SkillLabelRepository
	permChecker    *LabelPermissionChecker
	searchSyncer   SkillLabelSearchSyncer
}

// NewSkillLabelService creates a SkillLabelService.
func NewSkillLabelService(
	skillLabelRepo SkillLabelRepository,
	permChecker *LabelPermissionChecker,
	searchSyncer SkillLabelSearchSyncer,
) *SkillLabelService {
	if permChecker == nil {
		permChecker = NewLabelPermissionChecker()
	}
	return &SkillLabelService{
		skillLabelRepo: skillLabelRepo,
		permChecker:    permChecker,
		searchSyncer:   searchSyncer,
	}
}

// Assign assigns a label to a skill. Requires LABEL_ADMIN or SUPER_ADMIN.
func (svc *SkillLabelService) Assign(
	ctx context.Context,
	skillID, labelID int64,
	createdBy string,
	platformRoles map[string]bool,
) (*SkillLabel, error) {
	if !svc.permChecker.CanManageLabels(platformRoles) {
		return nil, fmt.Errorf("error.label.noPermission")
	}

	existing, _ := svc.skillLabelRepo.FindBySkillIDAndLabelID(ctx, skillID, labelID)
	if existing != nil {
		return existing, nil
	}

	sl := SkillLabel{
		SkillID:   skillID,
		LabelID:   labelID,
		CreatedBy: &createdBy,
		CreatedAt: time.Now(),
	}
	saved, err := svc.skillLabelRepo.Save(ctx, sl)
	if err != nil {
		return nil, fmt.Errorf("label: assign: %w", err)
	}

	if svc.searchSyncer != nil {
		_ = svc.searchSyncer.SyncSkillLabels(ctx, skillID)
	}
	return &saved, nil
}

// Remove removes a label assignment from a skill. Requires LABEL_ADMIN or SUPER_ADMIN.
func (svc *SkillLabelService) Remove(ctx context.Context, id int64, platformRoles map[string]bool) error {
	if !svc.permChecker.CanManageLabels(platformRoles) {
		return fmt.Errorf("error.label.noPermission")
	}

	sl, err := svc.skillLabelRepo.FindBySkillIDAndLabelID(ctx, 0, id)
	_ = sl
	if err != nil {
		return fmt.Errorf("label: find: %w", err)
	}

	if err := svc.skillLabelRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("label: remove: %w", err)
	}
	return nil
}

// GetForSkill returns all labels assigned to a skill.
func (svc *SkillLabelService) GetForSkill(ctx context.Context, skillID int64) ([]SkillLabel, error) {
	return svc.skillLabelRepo.FindBySkillID(ctx, skillID)
}

// GetForSkills returns labels for multiple skills.
func (svc *SkillLabelService) GetForSkills(ctx context.Context, skillIDs []int64) ([]SkillLabel, error) {
	return svc.skillLabelRepo.FindBySkillIDs(ctx, skillIDs)
}

// ---------------------------------------------------------------------------
// Permission checker
// ---------------------------------------------------------------------------

// LabelPermissionChecker authorizes label management operations.
type LabelPermissionChecker struct{}

// NewLabelPermissionChecker creates a LabelPermissionChecker.
func NewLabelPermissionChecker() *LabelPermissionChecker {
	return &LabelPermissionChecker{}
}

// CanManageLabels returns true if the caller has LABEL_ADMIN or SUPER_ADMIN role.
func (c *LabelPermissionChecker) CanManageLabels(platformRoles map[string]bool) bool {
	return platformRoles["LABEL_ADMIN"] || platformRoles["SUPER_ADMIN"]
}
