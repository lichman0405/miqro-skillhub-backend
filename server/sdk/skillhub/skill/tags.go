package skill

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// SkillTagService — tag management
// ---------------------------------------------------------------------------

// SkillTagService manages mutable tags that point to skill versions.
// Mirrors source com.iflytek.skillhub.domain.skill.service.SkillTagService.
type SkillTagService struct {
	tagRepo     SkillTagRepository
	versionRepo SkillVersionRepository
	skillRepo   SkillRepository
}

// NewSkillTagService creates a SkillTagService.  All three repositories
// are required — callers must not pass nil.
func NewSkillTagService(
	tagRepo SkillTagRepository,
	versionRepo SkillVersionRepository,
	skillRepo SkillRepository,
) *SkillTagService {
	return &SkillTagService{
		tagRepo:     tagRepo,
		versionRepo: versionRepo,
		skillRepo:   skillRepo,
	}
}

// CreateTag creates a tag pointing to a published version.
// The caller (actorID) must be the skill owner or hold ADMIN/OWNER role
// in the skill's namespace.
func (svc *SkillTagService) CreateTag(
	ctx context.Context,
	skillID int64,
	tagName string,
	versionStr string,
	actorID string,
	userNsRoles map[int64]string,
) (*SkillTag, error) {
	tagName = strings.TrimSpace(tagName)
	if tagName == "" {
		return nil, fmt.Errorf("skill: tag name required")
	}

	// Authorize: lookup skill and verify caller has lifecycle management rights.
	skill, err := svc.authorizeSkillLifecycle(ctx, skillID, actorID, userNsRoles)
	if err != nil {
		return nil, err
	}

	// Find the version (must be PUBLISHED).
	version, err := svc.versionRepo.FindBySkillIDAndVersion(ctx, skill.ID, versionStr)
	if err != nil || version == nil {
		return nil, fmt.Errorf("error.skill.version.notFound %s", versionStr)
	}
	if version.Status != "PUBLISHED" {
		return nil, fmt.Errorf("skill: only published versions can be tagged")
	}

	// Check for existing tag with same name (upsert).
	existing, _ := svc.tagRepo.FindBySkillIDAndTagName(ctx, skill.ID, tagName)
	if existing != nil {
		existing.VersionID = version.ID
		existing.UpdatedAt = time.Now()
		saved, err := svc.tagRepo.Save(ctx, *existing)
		if err != nil {
			return nil, err
		}
		return &saved, nil
	}

	now := time.Now()
	tag := SkillTag{
		SkillID:   skill.ID,
		TagName:   tagName,
		VersionID: version.ID,
		CreatedBy: &actorID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	saved, err := svc.tagRepo.Save(ctx, tag)
	if err != nil {
		return nil, err
	}
	return &saved, nil
}

// DeleteTag removes a tag by name.  The caller (actorID) must be the
// skill owner or hold ADMIN/OWNER role in the skill's namespace.
func (svc *SkillTagService) DeleteTag(
	ctx context.Context,
	skillID int64,
	tagName string,
	actorID string,
	userNsRoles map[int64]string,
) error {
	// Authorize: lookup skill and verify caller has lifecycle management rights.
	skill, err := svc.authorizeSkillLifecycle(ctx, skillID, actorID, userNsRoles)
	if err != nil {
		return err
	}

	tag, err := svc.tagRepo.FindBySkillIDAndTagName(ctx, skill.ID, tagName)
	if err != nil {
		return err
	}
	if tag == nil {
		return fmt.Errorf("error.skill.tag.notFound %s", tagName)
	}
	return svc.tagRepo.Delete(ctx, tag.ID)
}

// ListTags returns all tags for a skill.
func (svc *SkillTagService) ListTags(ctx context.Context, skillID int64) ([]SkillTag, error) {
	return svc.tagRepo.FindBySkillID(ctx, skillID)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// authorizeSkillLifecycle looks up the skill and verifies the actor has
// lifecycle management rights (owner, or namespace ADMIN/OWNER).
func (svc *SkillTagService) authorizeSkillLifecycle(
	ctx context.Context,
	skillID int64,
	actorID string,
	userNsRoles map[int64]string,
) (*Skill, error) {
	skill, err := svc.skillRepo.FindByID(ctx, skillID)
	if err != nil {
		return nil, fmt.Errorf("skill: find: %w", err)
	}
	if skill == nil {
		return nil, fmt.Errorf("error.skill.notFound")
	}
	if !canManageSkillLifecycle(*skill, actorID, userNsRoles) {
		return nil, fmt.Errorf("error.skill.access.denied")
	}
	return skill, nil
}
