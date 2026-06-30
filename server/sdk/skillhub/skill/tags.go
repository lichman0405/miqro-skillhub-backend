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

// NewSkillTagService creates a SkillTagService.
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
func (svc *SkillTagService) CreateTag(ctx context.Context, skillID int64, tagName string, versionStr string, actorID string, userNsRoles map[int64]string) (*SkillTag, error) {
	tagName = strings.TrimSpace(tagName)
	if tagName == "" {
		return nil, fmt.Errorf("skill: tag name required")
	}

	// Authorization check.
	if svc.skillRepo != nil {
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
	}

	// Find the version.
	version, err := svc.versionRepo.FindBySkillIDAndVersion(ctx, skillID, versionStr)
	if err != nil || version == nil {
		return nil, fmt.Errorf("error.skill.version.notFound %s", versionStr)
	}
	if version.Status != "PUBLISHED" {
		return nil, fmt.Errorf("skill: only published versions can be tagged")
	}

	// Check for existing tag with same name.
	existing, _ := svc.tagRepo.FindBySkillIDAndTagName(ctx, skillID, tagName)
	if existing != nil {
		// Update the existing tag.
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
		SkillID:   skillID,
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

// DeleteTag removes a tag by name.
func (svc *SkillTagService) DeleteTag(ctx context.Context, skillID int64, tagName string) error {
	tag, err := svc.tagRepo.FindBySkillIDAndTagName(ctx, skillID, tagName)
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
