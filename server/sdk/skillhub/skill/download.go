package skill

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/storage"
)

// ---------------------------------------------------------------------------
// Download result types
// ---------------------------------------------------------------------------

// DownloadResult wraps the download content and metadata.
type DownloadResult struct {
	Content        io.ReadCloser
	Filename       string
	ContentLength  int64
	ContentType    string
	PresignedURL   string
	FallbackBundle bool
}

// Close releases the underlying content stream.
func (r *DownloadResult) Close() error {
	if r.Content != nil {
		return r.Content.Close()
	}
	return nil
}

// ---------------------------------------------------------------------------
// SkillDownloadService
// ---------------------------------------------------------------------------

// SkillDownloadService delivers packaged skills to callers.
// Mirrors source com.iflytek.skillhub.domain.skill.service.SkillDownloadService.
type SkillDownloadService struct {
	nsRepo       namespace.NamespaceRepository
	skillRepo    SkillRepository
	versionRepo  SkillVersionRepository
	fileRepo     SkillFileRepository
	tagRepo      SkillTagRepository
	store        storage.Store
	visibility   *VisibilityChecker
}

// NewSkillDownloadService creates a SkillDownloadService.
func NewSkillDownloadService(
	nsRepo namespace.NamespaceRepository,
	skillRepo SkillRepository,
	versionRepo SkillVersionRepository,
	fileRepo SkillFileRepository,
	tagRepo SkillTagRepository,
	store storage.Store,
	visibility *VisibilityChecker,
) *SkillDownloadService {
	if visibility == nil {
		visibility = NewVisibilityChecker()
	}
	return &SkillDownloadService{
		nsRepo:     nsRepo,
		skillRepo:  skillRepo,
		versionRepo: versionRepo,
		fileRepo:   fileRepo,
		tagRepo:    tagRepo,
		store:      store,
		visibility: visibility,
	}
}

// DownloadLatest downloads the latest published version.
func (svc *SkillDownloadService) DownloadLatest(
	ctx context.Context,
	namespaceSlug, skillSlug, currentUserID string,
	userNsRoles map[int64]string,
) (*DownloadResult, error) {
	ns, skill, err := svc.resolveAndCheck(ctx, namespaceSlug, skillSlug, currentUserID, userNsRoles)
	if err != nil {
		return nil, err
	}
	if skill.LatestVersionID == nil {
		return nil, fmt.Errorf("error.skill.version.latest.unavailable %s", skillSlug)
	}
	version, err := svc.versionRepo.FindByID(ctx, *skill.LatestVersionID)
	if err != nil || version == nil {
		return nil, fmt.Errorf("error.skill.version.latest.notFound")
	}
	_ = ns
	return svc.downloadVersion(ctx, *skill, *version, currentUserID, userNsRoles)
}

// DownloadVersion downloads an explicit version.
func (svc *SkillDownloadService) DownloadVersion(
	ctx context.Context,
	namespaceSlug, skillSlug, versionStr, currentUserID string,
	userNsRoles map[int64]string,
) (*DownloadResult, error) {
	ns, skill, err := svc.resolveAndCheck(ctx, namespaceSlug, skillSlug, currentUserID, userNsRoles)
	if err != nil {
		return nil, err
	}
	version, err := svc.versionRepo.FindBySkillIDAndVersion(ctx, skill.ID, versionStr)
	if err != nil || version == nil {
		return nil, fmt.Errorf("error.skill.version.notFound %s", versionStr)
	}
	_ = ns
	return svc.downloadVersion(ctx, *skill, *version, currentUserID, userNsRoles)
}

// DownloadByTag downloads the version pointed to by a tag name.
func (svc *SkillDownloadService) DownloadByTag(
	ctx context.Context,
	namespaceSlug, skillSlug, tagName, currentUserID string,
	userNsRoles map[int64]string,
) (*DownloadResult, error) {
	ns, skill, err := svc.resolveAndCheck(ctx, namespaceSlug, skillSlug, currentUserID, userNsRoles)
	if err != nil {
		return nil, err
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
	_ = ns
	return svc.downloadVersion(ctx, *skill, *version, currentUserID, userNsRoles)
}

func (svc *SkillDownloadService) downloadVersion(
	ctx context.Context,
	skill Skill,
	version SkillVersion,
	currentUserID string,
	userNsRoles map[int64]string,
) (*DownloadResult, error) {
	if skill.Status != "ACTIVE" {
		return nil, fmt.Errorf("error.skill.status.notActive")
	}
	if !IsInstallable(version) {
		return nil, fmt.Errorf("error.skill.version.notDownloadable %s", version.Version)
	}
	result, err := svc.buildDownloadResult(ctx, skill, version)
	if err != nil {
		return nil, err
	}
	// Increment download count for published versions.
	_ = svc.skillRepo.IncrementDownloadCount(ctx, skill.ID)
	return result, nil
}

func (svc *SkillDownloadService) buildDownloadResult(ctx context.Context, skill Skill, version SkillVersion) (*DownloadResult, error) {
	bundleKey := fmt.Sprintf("packages/%d/%d/bundle.zip", skill.ID, version.ID)

	// Prefer bundle.
	exists, err := svc.store.Exists(ctx, bundleKey)
	if err == nil && exists {
		filename := buildDownloadFilename(skill, version)
		presigned, _ := svc.store.PresignedURL(ctx, bundleKey, 10*time.Minute, filename)
		rc, err := svc.store.GetObject(ctx, bundleKey)
		if err != nil {
			return nil, fmt.Errorf("skill: read bundle: %w", err)
		}
		return &DownloadResult{
			Content:        rc,
			Filename:       filename,
			ContentLength:  version.TotalSize, // approximate
			ContentType:    "application/zip",
			PresignedURL:   presigned,
			FallbackBundle: false,
		}, nil
	}

	// Fall back to per-file zip.
	return svc.buildBundleFromFiles(ctx, skill, version)
}

func (svc *SkillDownloadService) buildBundleFromFiles(ctx context.Context, skill Skill, version SkillVersion) (*DownloadResult, error) {
	files, err := svc.fileRepo.FindByVersionID(ctx, version.ID)
	if err != nil {
		return nil, fmt.Errorf("skill: list files: %w", err)
	}

	var availableFiles []SkillFile
	for _, f := range files {
		ok, err := svc.store.Exists(ctx, f.StorageKey)
		if err == nil && ok {
			availableFiles = append(availableFiles, f)
		}
	}
	if len(availableFiles) == 0 {
		return nil, fmt.Errorf("error.skill.bundle.notFound")
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, f := range availableFiles {
		rc, err := svc.store.GetObject(ctx, f.StorageKey)
		if err != nil {
			continue
		}
		entry, _ := w.Create(f.FilePath)
		io.Copy(entry, rc)
		rc.Close()
	}
	w.Close()

	result := buf.Bytes()
	return &DownloadResult{
		Content:        io.NopCloser(bytes.NewReader(result)),
		Filename:       buildDownloadFilename(skill, version),
		ContentLength:  int64(len(result)),
		ContentType:    "application/zip",
		FallbackBundle: true,
	}, nil
}

func (svc *SkillDownloadService) resolveAndCheck(
	ctx context.Context,
	namespaceSlug, skillSlug, currentUserID string,
	userNsRoles map[int64]string,
) (*namespace.Namespace, *Skill, error) {
	ns, err := svc.nsRepo.FindBySlug(ctx, namespaceSlug)
	if err != nil || ns == nil {
		return nil, nil, fmt.Errorf("error.namespace.slug.notFound %s", namespaceSlug)
	}

	skills, err := svc.skillRepo.FindByNamespaceIDAndSlug(ctx, ns.ID, skillSlug)
	if err != nil {
		return nil, nil, err
	}

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

	// Visibility check.
	if !svc.visibility.CanAccess(*skill, currentUserID, userNsRoles, nil) {
		return nil, nil, fmt.Errorf("error.skill.access.denied %s", skillSlug)
	}

	// Anonymous download only for PUBLIC.
	if currentUserID == "" && skill.Visibility != "PUBLIC" {
		return nil, nil, fmt.Errorf("error.skill.access.denied %s", skillSlug)
	}

	return ns, skill, nil
}

// ---------------------------------------------------------------------------
// Download helpers
// ---------------------------------------------------------------------------

var filenameSanitizeRe = regexp.MustCompile(`[\\/:*?"<>|]`)

func buildDownloadFilename(skill Skill, version SkillVersion) string {
	base := skill.DisplayName
	if base == "" {
		base = skill.Slug
	}
	sanitized := filenameSanitizeRe.ReplaceAllString(base, "-")
	sanitized = strings.Join(strings.Fields(sanitized), " ")
	sanitized = strings.TrimSpace(sanitized)
	if sanitized == "" {
		sanitized = "skill"
	}
	return fmt.Sprintf("%s-%s.zip", sanitized, version.Version)
}
