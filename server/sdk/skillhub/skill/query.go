package skill

import (
	"context"
	"fmt"
	"io"

	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/storage"
)

// ---------------------------------------------------------------------------
// SkillQueryService — read-side domain service
// ---------------------------------------------------------------------------

// SkillQueryService provides skill detail, version, and file inspection.
// Mirrors source com.iflytek.skillhub.domain.skill.service.SkillQueryService.
type SkillQueryService struct {
	nsRepo      namespace.NamespaceRepository
	skillRepo   SkillRepository
	versionRepo SkillVersionRepository
	fileRepo    SkillFileRepository
	tagRepo     SkillTagRepository
	store       storage.Store
	visibility  *VisibilityChecker
}

// NewSkillQueryService creates a SkillQueryService.
func NewSkillQueryService(
	nsRepo namespace.NamespaceRepository,
	skillRepo SkillRepository,
	versionRepo SkillVersionRepository,
	fileRepo SkillFileRepository,
	tagRepo SkillTagRepository,
	store storage.Store,
	visibility *VisibilityChecker,
) *SkillQueryService {
	if visibility == nil {
		visibility = NewVisibilityChecker()
	}
	return &SkillQueryService{
		nsRepo:     nsRepo,
		skillRepo:  skillRepo,
		versionRepo: versionRepo,
		fileRepo:   fileRepo,
		tagRepo:    tagRepo,
		store:      store,
		visibility: visibility,
	}
}

// ---------------------------------------------------------------------------
// Query result DTOs
// ---------------------------------------------------------------------------

// SkillDetail contains the visible fields of a skill.
type SkillDetail struct {
	ID                int64   `json:"id"`
	Slug              string  `json:"slug"`
	DisplayName       string  `json:"displayName"`
	OwnerID           string  `json:"ownerId"`
	Summary           string  `json:"summary"`
	Visibility        string  `json:"visibility"`
	Status            string  `json:"status"`
	DownloadCount     int64   `json:"downloadCount"`
	StarCount         int     `json:"starCount"`
	SubscriptionCount int     `json:"-"`
	RatingAvg         float64 `json:"ratingAvg"`
	RatingCount       int     `json:"-"`
	Hidden            bool    `json:"-"`
	NamespaceID       int64   `json:"-"`
	CanManage         bool    `json:"canManage"`
}

// VersionDetail contains the visible fields of a version.
type VersionDetail struct {
	ID                 int64  `json:"id"`
	Version            string `json:"version"`
	Status             string `json:"status"`
	FileCount          int    `json:"-"`
	TotalSize          int64  `json:"-"`
	PublishedAt        string `json:"publishedAt,omitempty"`
	ParsedMetadataJSON string `json:"-"`
	ManifestJSON       string `json:"-"`
}

// ---------------------------------------------------------------------------
// Query methods
// ---------------------------------------------------------------------------

// GetSkillDetail returns the skill detail for the given namespace+slug.
func (svc *SkillQueryService) GetSkillDetail(
	ctx context.Context,
	namespaceSlug, skillSlug, currentUserID string,
	userNsRoles map[int64]string,
	platformRoles map[string]bool,
) (*SkillDetail, error) {
	ns, skill, err := svc.resolveSkill(ctx, namespaceSlug, skillSlug, currentUserID)
	if err != nil {
		return nil, err
	}

	if ns.Status == "ARCHIVED" && userNsRoles[ns.ID] == "" {
		return nil, fmt.Errorf("error.namespace.archived %s", namespaceSlug)
	}

	if !svc.visibility.CanAccess(*skill, currentUserID, userNsRoles, platformRoles) {
		return nil, fmt.Errorf("error.skill.access.denied %s", skillSlug)
	}

	return &SkillDetail{
		ID:              skill.ID,
		Slug:            skill.Slug,
		DisplayName:     skill.DisplayName,
		OwnerID:         skill.OwnerID,
		Summary:         skill.Summary,
		Visibility:      skill.Visibility,
		Status:          skill.Status,
		DownloadCount:   skill.DownloadCount,
		StarCount:       skill.StarCount,
		SubscriptionCount: skill.SubscriptionCount,
		RatingAvg:       skill.RatingAvg,
		RatingCount:     skill.RatingCount,
		Hidden:          skill.Hidden,
		NamespaceID:     skill.NamespaceID,
		CanManage:       canManageSkillLifecycle(*skill, currentUserID, userNsRoles),
	}, nil
}

// GetSkillByID returns a skill by its numeric ID without namespace/slug resolution.
// Used by agent CI and other internal routes that reference skills by ID.
func (svc *SkillQueryService) GetSkillByID(ctx context.Context, skillID int64) (*Skill, error) {
	skill, err := svc.skillRepo.FindByID(ctx, skillID)
	if err != nil {
		return nil, fmt.Errorf("skill query: find skill by id: %w", err)
	}
	return skill, nil
}

// GetVersionDetail returns the metadata for a specific version.
func (svc *SkillQueryService) GetVersionDetail(
	ctx context.Context,
	namespaceSlug, skillSlug, versionStr, currentUserID string,
	userNsRoles map[int64]string,
) (*VersionDetail, error) {
	ns, skill, err := svc.resolveSkill(ctx, namespaceSlug, skillSlug, currentUserID)
	if err != nil {
		return nil, err
	}
	if !svc.visibility.CanAccess(*skill, currentUserID, userNsRoles, nil) {
		return nil, fmt.Errorf("error.skill.access.denied %s", skillSlug)
	}
	if ns.Status == "ARCHIVED" && userNsRoles[ns.ID] == "" {
		return nil, fmt.Errorf("error.namespace.archived %s", namespaceSlug)
	}

	version, err := svc.versionRepo.FindBySkillIDAndVersion(ctx, skill.ID, versionStr)
	if err != nil || version == nil {
		return nil, fmt.Errorf("error.skill.version.notFound %s", versionStr)
	}

	if version.Status != "PUBLISHED" && !canManageSkillLifecycle(*skill, currentUserID, userNsRoles) {
		return nil, fmt.Errorf("error.skill.version.notPublished %s", versionStr)
	}

	v := version
	var metadataJSON, manifestJSON string
	if v.ParsedMetadataJSON != nil {
		metadataJSON = *v.ParsedMetadataJSON
	}
	if v.ManifestJSON != nil {
		manifestJSON = *v.ManifestJSON
	}
	var publishedAt string
	if v.PublishedAt != nil {
		publishedAt = v.PublishedAt.Format("2006-01-02T15:04:05Z")
	}

	return &VersionDetail{
		ID:                 v.ID,
		Version:            v.Version,
		Status:             v.Status,
		FileCount:          v.FileCount,
		TotalSize:          v.TotalSize,
		PublishedAt:        publishedAt,
		ParsedMetadataJSON: metadataJSON,
		ManifestJSON:       manifestJSON,
	}, nil
}

// ListVersions returns versions for a skill, filtered by visibility.
func (svc *SkillQueryService) ListVersions(
	ctx context.Context,
	namespaceSlug, skillSlug, currentUserID string,
	userNsRoles map[int64]string,
) ([]SkillVersion, error) {
	_, skill, err := svc.resolveSkill(ctx, namespaceSlug, skillSlug, currentUserID)
	if err != nil {
		return nil, err
	}

	if canManageSkillLifecycle(*skill, currentUserID, userNsRoles) {
		return svc.versionRepo.FindBySkillID(ctx, skill.ID)
	}
	return svc.versionRepo.FindBySkillIDAndStatus(ctx, skill.ID, "PUBLISHED")
}

// ListFiles returns the file list for a version.
func (svc *SkillQueryService) ListFiles(
	ctx context.Context,
	namespaceSlug, skillSlug, versionStr, currentUserID string,
	userNsRoles map[int64]string,
) ([]SkillFile, error) {
	_, skill, err := svc.resolveSkill(ctx, namespaceSlug, skillSlug, currentUserID)
	if err != nil {
		return nil, err
	}

	version, err := svc.versionRepo.FindBySkillIDAndVersion(ctx, skill.ID, versionStr)
	if err != nil || version == nil {
		return nil, fmt.Errorf("error.skill.version.notFound %s", versionStr)
	}

	if version.Status != "PUBLISHED" && !canManageSkillLifecycle(*skill, currentUserID, userNsRoles) {
		return nil, fmt.Errorf("error.skill.version.notPublished %s", versionStr)
	}

	return svc.availableFiles(ctx, version.ID)
}

// GetFileContent returns the raw content of a single file.
func (svc *SkillQueryService) GetFileContent(
	ctx context.Context,
	namespaceSlug, skillSlug, versionStr, filePath, currentUserID string,
	userNsRoles map[int64]string,
) (io.ReadCloser, error) {
	_, skill, err := svc.resolveSkill(ctx, namespaceSlug, skillSlug, currentUserID)
	if err != nil {
		return nil, err
	}

	version, err := svc.versionRepo.FindBySkillIDAndVersion(ctx, skill.ID, versionStr)
	if err != nil || version == nil {
		return nil, fmt.Errorf("error.skill.version.notFound %s", versionStr)
	}

	if version.Status != "PUBLISHED" && !canManageSkillLifecycle(*skill, currentUserID, userNsRoles) {
		return nil, fmt.Errorf("error.skill.version.notPublished %s", versionStr)
	}

	files, err := svc.fileRepo.FindByVersionID(ctx, version.ID)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if f.FilePath == filePath {
			return svc.store.GetObject(ctx, f.StorageKey)
		}
	}
	return nil, fmt.Errorf("error.skill.file.notFound %s", filePath)
}

// ResolveVersion resolves a version selector (exact version, "latest", or tag).
func (svc *SkillQueryService) ResolveVersion(
	ctx context.Context,
	namespaceSlug, skillSlug, versionStr, tagName, currentUserID string,
	userNsRoles map[int64]string,
) (*SkillVersion, error) {
	_, skill, err := svc.resolveSkill(ctx, namespaceSlug, skillSlug, currentUserID)
	if err != nil {
		return nil, err
	}

	// Exact version.
	if versionStr != "" {
		version, err := svc.versionRepo.FindBySkillIDAndVersion(ctx, skill.ID, versionStr)
		if err != nil || version == nil {
			return nil, fmt.Errorf("error.skill.version.notFound %s", versionStr)
		}
		if version.Status != "PUBLISHED" {
			return nil, fmt.Errorf("error.skill.version.notPublished %s", versionStr)
		}
		return version, nil
	}

	// By tag.
	if tagName != "" {
		if stringsEqualFold(tagName, "latest") {
			if skill.LatestVersionID == nil {
				return nil, fmt.Errorf("error.skill.version.latest.unavailable %s", skillSlug)
			}
			version, err := svc.versionRepo.FindByID(ctx, *skill.LatestVersionID)
			if err != nil || version == nil {
				return nil, fmt.Errorf("error.skill.version.latest.notFound")
			}
			if version.Status != "PUBLISHED" {
				return nil, fmt.Errorf("error.skill.version.notPublished %s", version.Version)
			}
			return version, nil
		}

		tag, err := svc.tagRepo.FindBySkillIDAndTagName(ctx, skill.ID, tagName)
		if err != nil || tag == nil {
			return nil, fmt.Errorf("error.skill.tag.notFound %s", tagName)
		}
		if tag.VersionID == 0 {
			return nil, fmt.Errorf("error.skill.tag.version.missing %s", tagName)
		}
		version, err := svc.versionRepo.FindByID(ctx, tag.VersionID)
		if err != nil || version == nil {
			return nil, fmt.Errorf("error.skill.tag.version.notFound %s", tagName)
		}
		return version, nil
	}

	// Default to latest.
	if skill.LatestVersionID == nil {
		return nil, fmt.Errorf("error.skill.version.latest.unavailable %s", skillSlug)
	}
	version, err := svc.versionRepo.FindByID(ctx, *skill.LatestVersionID)
	if err != nil || version == nil {
		return nil, fmt.Errorf("error.skill.version.latest.notFound")
	}
	if version.Status != "PUBLISHED" {
		return nil, fmt.Errorf("error.skill.version.notPublished %s", version.Version)
	}
	return version, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (svc *SkillQueryService) resolveSkill(ctx context.Context, namespaceSlug, skillSlug, currentUserID string) (*namespace.Namespace, *Skill, error) {
	ns, err := svc.nsRepo.FindBySlug(ctx, namespaceSlug)
	if err != nil || ns == nil {
		return nil, nil, fmt.Errorf("error.namespace.slug.notFound %s", namespaceSlug)
	}

	skills, err := svc.skillRepo.FindByNamespaceIDAndSlug(ctx, ns.ID, skillSlug)
	if err != nil {
		return nil, nil, err
	}

	// Prefer the current user's skill.
	var skill *Skill
	for i := range skills {
		if skills[i].OwnerID == currentUserID {
			skill = &skills[i]
			break
		}
	}
	if skill == nil && len(skills) > 0 {
		skill = &skills[0]
	}
	if skill == nil {
		return nil, nil, fmt.Errorf("error.skill.notFound %s", skillSlug)
	}

	return ns, skill, nil
}

func (svc *SkillQueryService) availableFiles(ctx context.Context, versionID int64) ([]SkillFile, error) {
	files, err := svc.fileRepo.FindByVersionID(ctx, versionID)
	if err != nil {
		return nil, err
	}
	var available []SkillFile
	for _, f := range files {
		ok, err := svc.store.Exists(ctx, f.StorageKey)
		if err == nil && ok {
			available = append(available, f)
		}
	}
	return available, nil
}

func stringsEqualFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
