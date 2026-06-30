package skill

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/packagekit"
	"miqro-skillhub/server/sdk/skillhub/storage"
	"miqro-skillhub/server/sdk/skillhub/uow"
)

const autoVersionLayout = "20060102.150405"

// ---------------------------------------------------------------------------
// Publish result types
// ---------------------------------------------------------------------------

// PublishResult is returned after a successful publish.
type PublishResult struct {
	SkillID  int64
	Slug     string
	Version  SkillVersion
}

// DryRunResult is the outcome of a dry-run validation (validateOnly).
type DryRunResult struct {
	Valid           bool
	Errors          []string
	Warnings        []string
	ResolvedSlug    string
	ResolvedVersion string
}

// ---------------------------------------------------------------------------
// Pre-publish validator interface (Phase 06 will wire real implementations)
// ---------------------------------------------------------------------------

// PrePublishValidator performs additional checks (e.g. credential scanning)
// before a skill is published.  Phase 05 ships a no-op implementation.
type PrePublishValidator interface {
	Validate(ctx context.Context, entries []packagekit.PackageEntry, metadata *packagekit.SkillMetadata, publisherID string, namespaceID int64) *packagekit.ValidationResult
}

// NoOpPrePublishValidator is a pre-publish validator that always passes.
type NoOpPrePublishValidator struct{}

func (NoOpPrePublishValidator) Validate(_ context.Context, _ []packagekit.PackageEntry, _ *packagekit.SkillMetadata, _ string, _ int64) *packagekit.ValidationResult {
	return &packagekit.ValidationResult{}
}

// ---------------------------------------------------------------------------
// Scanner dependency (Phase 06+ will wire real implementation)
// ---------------------------------------------------------------------------

// SecurityScanner is the interface for triggering security scans.
type SecurityScanner interface {
	IsEnabled() bool
}

// NoOpScanner is a scanner that is always disabled.
type NoOpScanner struct{}

func (NoOpScanner) IsEnabled() bool { return false }

// ---------------------------------------------------------------------------
// SkillPublishService
// ---------------------------------------------------------------------------

// SkillPublishService publishes packaged skill artifacts into persisted
// skill and version records.
// Mirrors source com.iflytek.skillhub.domain.skill.service.SkillPublishService.
type SkillPublishService struct {
	nsRepo          namespace.NamespaceRepository
	nsMemberRepo    namespace.NamespaceMemberRepository
	skillRepo       SkillRepository
	versionRepo     SkillVersionRepository
	fileRepo        SkillFileRepository
	store           storage.Store
	validator       *packagekit.SkillPackageValidator
	metadataParser  *packagekit.SkillMetadataParser
	preValidator    PrePublishValidator
	scanner         SecurityScanner
	transactor      uow.Transactor
}

// NewSkillPublishService creates a SkillPublishService wired with its dependencies.
func NewSkillPublishService(
	nsRepo namespace.NamespaceRepository,
	nsMemberRepo namespace.NamespaceMemberRepository,
	skillRepo SkillRepository,
	versionRepo SkillVersionRepository,
	fileRepo SkillFileRepository,
	store storage.Store,
	validator *packagekit.SkillPackageValidator,
	metadataParser *packagekit.SkillMetadataParser,
	preValidator PrePublishValidator,
	scanner SecurityScanner,
	transactor uow.Transactor,
) *SkillPublishService {
	if preValidator == nil {
		preValidator = NoOpPrePublishValidator{}
	}
	if scanner == nil {
		scanner = NoOpScanner{}
	}
	return &SkillPublishService{
		nsRepo:         nsRepo,
		nsMemberRepo:   nsMemberRepo,
		skillRepo:      skillRepo,
		versionRepo:    versionRepo,
		fileRepo:       fileRepo,
		store:          store,
		validator:      validator,
		metadataParser: metadataParser,
		preValidator:   preValidator,
		scanner:        scanner,
		transactor:     transactor,
	}
}

// ---------------------------------------------------------------------------
// Dry-run validation
// ---------------------------------------------------------------------------

// ValidateOnly validates a package without persisting anything.
// Mirrors source SkillPublishService.validateOnly.
func (svc *SkillPublishService) ValidateOnly(
	ctx context.Context,
	namespaceSlug string,
	entries []packagekit.PackageEntry,
	publisherID string,
	visibility string,
	platformRoles map[string]bool,
) (*DryRunResult, error) {
	var errors []string
	var warnings []string
	var resolvedSlug, resolvedVersion string

	// 1. Find namespace.
	ns, err := svc.nsRepo.FindBySlug(ctx, namespaceSlug)
	if err != nil {
		return nil, fmt.Errorf("skill: find namespace: %w", err)
	}
	if ns == nil {
		errors = append(errors, "Namespace not found: "+namespaceSlug)
		return &DryRunResult{Valid: false, Errors: errors, Warnings: warnings}, nil
	}
	if ns.Status == "FROZEN" {
		errors = append(errors, "Namespace is frozen: "+namespaceSlug)
	}
	if ns.Status == "ARCHIVED" {
		errors = append(errors, "Namespace is archived: "+namespaceSlug)
	}

	// 2. Check membership.
	isSuperAdmin := platformRoles["SUPER_ADMIN"]
	if !isSuperAdmin {
		member, _ := svc.nsMemberRepo.FindByNamespaceAndUser(ctx, ns.ID, publisherID)
		if member == nil {
			errors = append(errors, "Publisher is not a member of namespace: "+namespaceSlug)
		}
	}

	// 3. Package validation.
	pkgResult := svc.validator.Validate(entries)
	errors = append(errors, pkgResult.Errors...)
	warnings = append(warnings, pkgResult.Warnings...)

	if !pkgResult.Passed() {
		return &DryRunResult{Valid: false, Errors: errors, Warnings: warnings, ResolvedSlug: resolvedSlug, ResolvedVersion: resolvedVersion}, nil
	}

	// 4. Parse SKILL.md.
	var skillMd *packagekit.PackageEntry
	for i := range entries {
		if entries[i].Path == "SKILL.md" {
			skillMd = &entries[i]
			break
		}
	}
	if skillMd == nil {
		errors = append(errors, "Missing required file: SKILL.md at root")
		return &DryRunResult{Valid: false, Errors: errors, Warnings: warnings}, nil
	}

	metadata, err := svc.metadataParser.Parse(string(skillMd.Content))
	if err != nil {
		errors = append(errors, "Invalid SKILL.md: "+err.Error())
		return &DryRunResult{Valid: false, Errors: errors, Warnings: warnings}, nil
	}

	if metadata.Version == "" {
		resolvedVersion = time.Now().Format(autoVersionLayout)
	} else {
		resolvedVersion = metadata.Version
	}

	// Slug is derived from metadata name.
	resolvedSlug, slugErr := namespace.Slugify(metadata.Name)
	if slugErr != nil {
		errors = append(errors, "Invalid skill name for slug generation: "+slugErr.Error())
		return &DryRunResult{Valid: false, Errors: errors, Warnings: warnings, ResolvedSlug: resolvedSlug, ResolvedVersion: resolvedVersion}, nil
	}

	// 5. Pre-publish validation.
	preResult := svc.preValidator.Validate(ctx, entries, metadata, publisherID, ns.ID)
	errors = append(errors, preResult.Errors...)
	warnings = append(warnings, preResult.Warnings...)

	// 6. Slug conflict checks.
	if resolvedSlug != "" && len(errors) == 0 {
		existingSkills, err := svc.skillRepo.FindByNamespaceIDAndSlug(ctx, ns.ID, resolvedSlug)
		if err != nil {
			return nil, fmt.Errorf("skill: find existing: %w", err)
		}
		for _, existing := range existingSkills {
			if existing.OwnerID == publisherID {
				if existing.Status == "ARCHIVED" {
					errors = append(errors, "Cannot publish to archived skill: "+resolvedSlug)
				}
				if resolvedVersion != "" {
					ver, _ := svc.versionRepo.FindBySkillIDAndVersion(ctx, existing.ID, resolvedVersion)
					if ver != nil && ver.Status == "PUBLISHED" {
						errors = append(errors, "Version already published: "+resolvedVersion)
					}
				}
			} else {
				published, _ := svc.versionRepo.FindBySkillIDAndStatus(ctx, existing.ID, "PUBLISHED")
				if len(published) > 0 {
					errors = append(errors, "Name conflict: slug \""+resolvedSlug+"\" is already published by another user")
					break
				}
			}
		}
	}

	valid := len(errors) == 0 && len(warnings) == 0
	return &DryRunResult{
		Valid:           valid,
		Errors:          errors,
		Warnings:        warnings,
		ResolvedSlug:    resolvedSlug,
		ResolvedVersion: resolvedVersion,
	}, nil
}

// ---------------------------------------------------------------------------
// Main publish flow
// ---------------------------------------------------------------------------

// Publish publishes an extracted package into the target namespace.
func (svc *SkillPublishService) Publish(
	ctx context.Context,
	namespaceSlug string,
	entries []packagekit.PackageEntry,
	publisherID string,
	visibility string,
	platformRoles map[string]bool,
	confirmWarnings bool,
) (*PublishResult, error) {
	return svc.publishInternal(ctx, namespaceSlug, entries, publisherID, visibility, platformRoles, confirmWarnings, false, false)
}

// publishInternal performs the full publish flow.
func (svc *SkillPublishService) publishInternal(
	ctx context.Context,
	namespaceSlug string,
	entries []packagekit.PackageEntry,
	publisherID string,
	visibility string,
	platformRoles map[string]bool,
	confirmWarnings bool,
	forceAutoPublish bool,
	bypassMembershipCheck bool,
) (*PublishResult, error) {

	// 1. Find namespace.
	ns, err := svc.nsRepo.FindBySlug(ctx, namespaceSlug)
	if err != nil {
		return nil, fmt.Errorf("skill: find namespace: %w", err)
	}
	if ns == nil {
		return nil, fmt.Errorf("skill: namespace not found: %s", namespaceSlug)
	}
	if err := assertNamespaceWritable(ns); err != nil {
		return nil, err
	}

	isSuperAdmin := platformRoles["SUPER_ADMIN"]

	// 2. Check publisher membership.
	if !isSuperAdmin && !bypassMembershipCheck {
		member, _ := svc.nsMemberRepo.FindByNamespaceAndUser(ctx, ns.ID, publisherID)
		if member == nil {
			return nil, fmt.Errorf("skill: publisher not member of namespace: %s", namespaceSlug)
		}
	}

	// 3. Validate package.
	pkgResult := svc.validator.Validate(entries)
	if !pkgResult.Passed() {
		return nil, fmt.Errorf("error.skill.publish.package.invalid %s", strings.Join(pkgResult.Errors, ", "))
	}

	// 4. Parse SKILL.md.
	var skillMd *packagekit.PackageEntry
	for i := range entries {
		if entries[i].Path == "SKILL.md" {
			skillMd = &entries[i]
			break
		}
	}
	if skillMd == nil {
		return nil, fmt.Errorf("skill: SKILL.md not found")
	}

	metadata, err := svc.metadataParser.Parse(string(skillMd.Content))
	if err != nil {
		return nil, fmt.Errorf("skill: parse metadata: %w", err)
	}
	if metadata.Version == "" {
		metadata.Version = time.Now().Format(autoVersionLayout)
	}
	skillSlug, err := namespace.Slugify(metadata.Name)
	if err != nil {
		return nil, fmt.Errorf("skill: slugify: %w", err)
	}

	// 5. Pre-publish validation.
	preResult := svc.preValidator.Validate(ctx, entries, metadata, publisherID, ns.ID)
	if !preResult.Passed() {
		return nil, fmt.Errorf("error.skill.publish.precheck.failed %s", strings.Join(preResult.Errors, ", "))
	}
	publishWarnings := append(append([]string{}, pkgResult.Warnings...), preResult.Warnings...)
	if !confirmWarnings && len(publishWarnings) > 0 {
		return nil, fmt.Errorf("error.skill.publish.precheck.confirmRequired %s",
			strings.Join(publishWarnings, ", "))
	}

	// 6. Find or create skill record.
	existingSkills, err := svc.skillRepo.FindByNamespaceIDAndSlug(ctx, ns.ID, skillSlug)
	if err != nil {
		return nil, fmt.Errorf("skill: find existing: %w", err)
	}
	for _, existing := range existingSkills {
		if existing.OwnerID != publisherID {
			published, _ := svc.versionRepo.FindBySkillIDAndStatus(ctx, existing.ID, "PUBLISHED")
			if len(published) > 0 {
				return nil, fmt.Errorf("error.skill.publish.nameConflict %s", skillSlug)
			}
		}
	}

	skill, err := svc.skillRepo.FindByNamespaceIDSlugOwner(ctx, ns.ID, skillSlug, publisherID)
	if err != nil {
		return nil, fmt.Errorf("skill: find by owner: %w", err)
	}
	if skill == nil {
		newSkill := Skill{
			NamespaceID: ns.ID,
			Slug:        skillSlug,
			OwnerID:     publisherID,
			Visibility:  visibility,
			Status:      "ACTIVE",
			CreatedBy:   &publisherID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		savedSkill, err := svc.skillRepo.Save(ctx, newSkill)
		if err != nil {
			return nil, fmt.Errorf("skill: save: %w", err)
		}
		skill = &savedSkill
	}

	if skill.Status == "ARCHIVED" {
		return nil, fmt.Errorf("error.skill.publish.archived %s", skillSlug)
	}

	// 6c. Auto-withdraw pending review versions (Phase 06 concern, but structure exists).
	// For Phase 05, we skip review task management but prepare the version status flow.

	// 7. Check version doesn't already exist.
	existingVer, _ := svc.versionRepo.FindBySkillIDAndVersion(ctx, skill.ID, metadata.Version)
	if existingVer != nil {
		if existingVer.Status == "PUBLISHED" {
			return nil, fmt.Errorf("error.skill.version.exists %s", metadata.Version)
		}
		// For non-published duplicates, delete and replace (storage handled after commit).
		_ = svc.fileRepo.DeleteByVersionID(ctx, existingVer.ID)
		_ = svc.versionRepo.Delete(ctx, existingVer.ID)
	}

	// 8. Create SkillVersion.
	autoPublish := forceAutoPublish || isSuperAdmin
	version := SkillVersion{
		SkillID:  skill.ID,
		Version:  metadata.Version,
		Status:   resolveInitialStatus(autoPublish, visibility),
		CreatedBy: publisherID,
		CreatedAt: time.Now(),
	}
	if autoPublish {
		now := time.Now()
		version.PublishedAt = &now
	}

	// Serialize metadata.
	metadataJSON := mustMarshalJSON(metadata)
	manifestJSON := mustMarshalJSON(buildManifest(entries))
	version.ParsedMetadataJSON = &metadataJSON
	version.ManifestJSON = &manifestJSON

	savedVersion, err := svc.versionRepo.Save(ctx, version)
	if err != nil {
		return nil, fmt.Errorf("skill: save version: %w", err)
	}

	// 9. Upload each file and compute SHA-256.
	var skillFiles []SkillFile
	var totalSize int64

	for _, entry := range entries {
		storageKey := fmt.Sprintf("skills/%d/%d/%s", skill.ID, savedVersion.ID, entry.Path)

		if err := svc.store.PutObject(ctx, storageKey, bytes.NewReader(entry.Content), entry.Size, entry.ContentType); err != nil {
			return nil, fmt.Errorf("skill: upload %s: %w", entry.Path, err)
		}

		hash := sha256.Sum256(entry.Content)
		sha256Hex := hex.EncodeToString(hash[:])

		skillFiles = append(skillFiles, SkillFile{
			VersionID:   savedVersion.ID,
			FilePath:    entry.Path,
			FileSize:    entry.Size,
			ContentType: entry.ContentType,
			SHA256:      sha256Hex,
			StorageKey:  storageKey,
			CreatedAt:   time.Now(),
		})
		totalSize += entry.Size
	}

	// 10. Save file records.
	if _, err := svc.fileRepo.SaveAll(ctx, skillFiles); err != nil {
		return nil, fmt.Errorf("skill: save files: %w", err)
	}

	// 10.5 Build and upload bundle.
	bundleZip := buildBundle(entries)
	bundleKey := fmt.Sprintf("packages/%d/%d/bundle.zip", skill.ID, savedVersion.ID)
	if err := svc.store.PutObject(ctx, bundleKey, bytes.NewReader(bundleZip), int64(len(bundleZip)), "application/zip"); err != nil {
		return nil, fmt.Errorf("skill: upload bundle: %w", err)
	}

	// 11. Update version stats.
	savedVersion.FileCount = len(skillFiles)
	savedVersion.TotalSize = totalSize
	savedVersion.BundleReady = true
	savedVersion.DownloadReady = len(skillFiles) > 0
	savedVersion, err = svc.versionRepo.Save(ctx, savedVersion)
	if err != nil {
		return nil, fmt.Errorf("skill: update version stats: %w", err)
	}

	// 12. Update skill metadata.
	skill.DisplayName = metadata.Name
	skill.Summary = metadata.Description
	if autoPublish || visibility == "PRIVATE" {
		skill.LatestVersionID = &savedVersion.ID
		skill.Visibility = visibility
	}
	skill.UpdatedBy = &publisherID
	skill.UpdatedAt = time.Now()
	if _, err := svc.skillRepo.Save(ctx, *skill); err != nil {
		return nil, fmt.Errorf("skill: save skill: %w", err)
	}

	// Phase 05 note: review task creation and security scan triggering are Phase 06 concerns.

	return &PublishResult{
		SkillID: skill.ID,
		Slug:    skill.Slug,
		Version: savedVersion,
	}, nil
}

// ---------------------------------------------------------------------------
// Rerelease
// ---------------------------------------------------------------------------

// RereleasePublishedVersion rebuilds a new version from an already published
// version by copying its stored files and rewriting the SKILL.md version field.
// Mirrors source SkillPublishService.rereleasePublishedVersion.
// Phase 05 provides the foundation; full implementation comes in Phase 06 when
// lifecycle management permissions are fully wired.
func (svc *SkillPublishService) RereleasePublishedVersion(
	ctx context.Context,
	skillID int64,
	sourceVersion string,
	targetVersion string,
	publisherID string,
	userNsRoles map[int64]string,
	confirmWarnings bool,
) (*PublishResult, error) {
	skill, err := svc.skillRepo.FindByID(ctx, skillID)
	if err != nil {
		return nil, fmt.Errorf("skill: find: %w", err)
	}
	if skill == nil {
		return nil, fmt.Errorf("skill: not found")
	}

	// Check lifecycle management permission.
	if !canManageSkillLifecycle(*skill, publisherID, userNsRoles) {
		return nil, fmt.Errorf("skill: lifecycle permission denied")
	}

	publishedVer, err := svc.versionRepo.FindBySkillIDAndVersion(ctx, skillID, sourceVersion)
	if err != nil {
		return nil, fmt.Errorf("skill: find source version: %w", err)
	}
	if publishedVer == nil || publishedVer.Status != "PUBLISHED" {
		return nil, fmt.Errorf("skill: source version not published: %s", sourceVersion)
	}

	dup, _ := svc.versionRepo.FindBySkillIDAndVersion(ctx, skillID, targetVersion)
	if dup != nil {
		return nil, fmt.Errorf("skill: target version already exists: %s", targetVersion)
	}

	// Rebuild entries from stored files with version rewrite.
	entries, err := svc.rebuildEntriesForRerelease(ctx, skillID, publishedVer.ID, targetVersion)
	if err != nil {
		return nil, fmt.Errorf("skill: rebuild entries: %w", err)
	}

	ns, err := svc.nsRepo.FindByID(ctx, skill.NamespaceID)
	if err != nil {
		return nil, fmt.Errorf("skill: find namespace: %w", err)
	}
	if ns == nil {
		return nil, fmt.Errorf("skill: namespace not found")
	}

	return svc.publishInternal(ctx, ns.Slug, entries, publisherID, skill.Visibility, nil, confirmWarnings, false, true)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func assertNamespaceWritable(ns *namespace.Namespace) error {
	if ns.Status == "FROZEN" {
		return fmt.Errorf("error.namespace.frozen %s", ns.Slug)
	}
	if ns.Status == "ARCHIVED" {
		return fmt.Errorf("error.namespace.archived %s", ns.Slug)
	}
	return nil
}

func resolveInitialStatus(autoPublish bool, visibility string) string {
	if autoPublish {
		return "PUBLISHED"
	}
	if visibility == "PRIVATE" {
		return "UPLOADED"
	}
	return "PENDING_REVIEW"
}

func canManageSkillLifecycle(skill Skill, actorID string, userNsRoles map[int64]string) bool {
	if skill.OwnerID == actorID {
		return true
	}
	role := userNsRoles[skill.NamespaceID]
	return role == "OWNER" || role == "ADMIN"
}

func buildBundle(entries []packagekit.PackageEntry) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, e := range entries {
		f, _ := w.Create(e.Path)
		f.Write(e.Content)
	}
	w.Close()
	return buf.Bytes()
}

func buildManifest(entries []packagekit.PackageEntry) []map[string]any {
	var m []map[string]any
	for _, e := range entries {
		m = append(m, map[string]any{
			"path":        e.Path,
			"size":        e.Size,
			"contentType": e.ContentType,
		})
	}
	return m
}

func mustMarshalJSON(v any) string {
	s, err := marshalSimpleJSON(v)
	if err != nil {
		return "{}"
	}
	return s
}

// marshalSimpleJSON is a minimal JSON marshaler for metadata/manifest.
func marshalSimpleJSON(v any) (string, error) {
	switch val := v.(type) {
	case *packagekit.SkillMetadata:
		return marshalSkillMetadata(val), nil
	case []map[string]any:
		return marshalManifest(val), nil
	default:
		return "{}", nil
	}
}

func marshalSkillMetadata(m *packagekit.SkillMetadata) string {
	// Build a simple JSON string manually to avoid importing encoding/json
	// with interface{} values.
	parts := []string{
		fmt.Sprintf(`"name":%q`, m.Name),
		fmt.Sprintf(`"description":%q`, m.Description),
		fmt.Sprintf(`"version":%q`, m.Version),
		fmt.Sprintf(`"body":%q`, m.Body),
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func marshalManifest(entries []map[string]any) string {
	var parts []string
	for _, e := range entries {
		p := fmt.Sprintf(`{"path":%q,"size":%v,"contentType":%q}`,
			e["path"], e["size"], e["contentType"])
		parts = append(parts, p)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func (svc *SkillPublishService) rebuildEntriesForRerelease(ctx context.Context, skillID, versionID int64, targetVersion string) ([]packagekit.PackageEntry, error) {
	files, err := svc.fileRepo.FindByVersionID(ctx, versionID)
	if err != nil {
		return nil, err
	}
	var entries []packagekit.PackageEntry
	for _, f := range files {
		rc, err := svc.store.GetObject(ctx, f.StorageKey)
		if err != nil {
			return nil, fmt.Errorf("skill: get object %s: %w", f.StorageKey, err)
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			rc.Close()
			return nil, err
		}
		rc.Close()

		content := buf.Bytes()
		if f.FilePath == "SKILL.md" {
			content = rewriteSkillMdVersion(content, targetVersion)
		}

		contentType := f.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		entries = append(entries, packagekit.PackageEntry{
			Path:        f.FilePath,
			Content:     content,
			Size:        int64(len(content)),
			ContentType: contentType,
		})
	}
	return entries, nil
}

func rewriteSkillMdVersion(content []byte, targetVersion string) []byte {
	parser := packagekit.NewSkillMetadataParser()
	metadata, err := parser.Parse(string(content))
	if err != nil {
		return content
	}
	// Rewrite the frontmatter with the new version and rebuild SKILL.md.
	var yamlLines []string
	fm := metadata.Frontmatter
	if fm == nil {
		fm = make(map[string]any)
	}
	fm["version"] = targetVersion
	yamlLines = append(yamlLines, "---")
	for k, v := range fm {
		yamlLines = append(yamlLines, fmt.Sprintf("%s: %v", k, v))
	}
	yamlLines = append(yamlLines, "---")
	yamlLines = append(yamlLines, metadata.Body)
	return []byte(strings.Join(yamlLines, "\n") + "\n")
}
