package skill

import (
	"context"
	"fmt"

	"miqro-skillhub/server/sdk/skillhub/storage"
)

// ---------------------------------------------------------------------------
// SkillHardDeleteService — permanent skill deletion
// ---------------------------------------------------------------------------

// SkillHardDeleteService permanently deletes a skill and all its artifacts.
// Mirrors source com.iflytek.skillhub.domain.skill.service.SkillHardDeleteService.
//
// Phase 05 provides the core delete logic. Review/promotion/scan/audit
// cleanup hooks are wired in Phase 06-07 when those packages exist.
type SkillHardDeleteService struct {
	skillRepo       SkillRepository
	versionRepo     SkillVersionRepository
	fileRepo        SkillFileRepository
	tagRepo         SkillTagRepository
	versionStatsRepo SkillVersionStatsRepository
	compRepo        SkillStorageDeletionCompensationRepository
	store           storage.Store
}

// NewSkillHardDeleteService creates a SkillHardDeleteService.
func NewSkillHardDeleteService(
	skillRepo SkillRepository,
	versionRepo SkillVersionRepository,
	fileRepo SkillFileRepository,
	tagRepo SkillTagRepository,
	versionStatsRepo SkillVersionStatsRepository,
	compRepo SkillStorageDeletionCompensationRepository,
	store storage.Store,
) *SkillHardDeleteService {
	return &SkillHardDeleteService{
		skillRepo:       skillRepo,
		versionRepo:     versionRepo,
		fileRepo:        fileRepo,
		tagRepo:         tagRepo,
		versionStatsRepo: versionStatsRepo,
		compRepo:        compRepo,
		store:           store,
	}
}

// HardDelete permanently removes a skill and all associated data.
// The caller must be the skill owner or hold ADMIN/OWNER role in the skill's
// namespace.  userNsRoles maps namespaceID → role string.
//
// Storage objects are deleted after the DB commit with compensation on failure.
func (svc *SkillHardDeleteService) HardDelete(ctx context.Context, skillID int64, namespaceSlug, actorID string, userNsRoles map[int64]string) error {
	skill, err := svc.skillRepo.FindByID(ctx, skillID)
	if err != nil {
		return fmt.Errorf("skill: find: %w", err)
	}
	if skill == nil {
		return fmt.Errorf("error.skill.notFound")
	}

	// Authorization: only the skill owner or a namespace ADMIN/OWNER may hard-delete.
	if !canManageSkillLifecycle(*skill, actorID, userNsRoles) {
		return fmt.Errorf("error.skill.access.denied")
	}

	// Collect all storage keys from all versions.
	versions, err := svc.versionRepo.FindBySkillID(ctx, skill.ID)
	if err != nil {
		return fmt.Errorf("skill: list versions: %w", err)
	}

	var storageKeys []string
	for _, version := range versions {
		files, err := svc.fileRepo.FindByVersionID(ctx, version.ID)
		if err != nil {
			return fmt.Errorf("skill: list files: %w", err)
		}
		for _, f := range files {
			if f.StorageKey != "" {
				storageKeys = append(storageKeys, f.StorageKey)
			}
		}
		storageKeys = append(storageKeys, fmt.Sprintf("packages/%d/%d/bundle.zip", skill.ID, version.ID))
	}

	// Nullify latest version pointer first.
	skill.LatestVersionID = nil
	skill.UpdatedBy = &actorID
	savedSkill, err := svc.skillRepo.Save(ctx, *skill)
	if err != nil {
		return fmt.Errorf("skill: save: %w", err)
	}
	skill = &savedSkill

	// Delete related data.
	if err := svc.tagRepo.DeleteBySkillID(ctx, skill.ID); err != nil {
		return fmt.Errorf("skill: delete tags: %w", err)
	}
	if svc.versionStatsRepo != nil {
		_ = svc.versionStatsRepo.DeleteBySkillID(ctx, skill.ID)
	}

	// Delete files and versions.
	for _, version := range versions {
		_ = svc.fileRepo.DeleteByVersionID(ctx, version.ID)
		_ = svc.versionRepo.Delete(ctx, version.ID)
	}

	// Delete the skill.
	if err := svc.skillRepo.Delete(ctx, skill.ID); err != nil {
		return fmt.Errorf("skill: delete: %w", err)
	}

	// Delete storage objects after DB commit (best-effort with compensation).
	svc.deleteStorageWithCompensation(skill.ID, namespaceSlug, skill.Slug, storageKeys)

	return nil
}

func (svc *SkillHardDeleteService) deleteStorageWithCompensation(skillID int64, namespaceSlug, slug string, keys []string) {
	if len(keys) == 0 {
		return
	}
	if err := svc.store.DeleteObjects(context.Background(), keys); err != nil && svc.compRepo != nil {
		_ = recordCompensation(svc.compRepo, skillID, namespaceSlug, slug, keys, err)
	}
}

func recordCompensation(compRepo SkillStorageDeletionCompensationRepository, skillID int64, namespaceSlug, slug string, keys []string, err error) error {
	comp := SkillStorageDeletionCompensation{
		SkillID:   &skillID,
		Namespace: namespaceSlug,
		Slug:      slug,
		StorageKeysJSON: fmt.Sprintf("%q", keys),
		Status:    "PENDING",
	}
	_, saveErr := compRepo.Save(context.Background(), comp)
	return saveErr
}
