package skill

import (
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/packagekit"
	"miqro-skillhub/server/sdk/skillhub/storage"
	"miqro-skillhub/server/sdk/skillhub/uow"
)

// ---------------------------------------------------------------------------
// Service — top-level facade assembling all skill sub-services
// ---------------------------------------------------------------------------

// Service is the main skill SDK service, assembling all sub-services.
type Service struct {
	Publish  *SkillPublishService
	Download *SkillDownloadService
	Query    *SkillQueryService
	Tags     *SkillTagService
	Delete   *SkillHardDeleteService
	Visibility *VisibilityChecker
	Installability *SkillInstallability
}

// ServiceConfig holds the dependencies for creating a skill Service.
type ServiceConfig struct {
	// Repository dependencies.
	NamespaceRepo    namespace.NamespaceRepository
	NamespaceMemberRepo namespace.NamespaceMemberRepository
	SkillRepo        SkillRepository
	VersionRepo      SkillVersionRepository
	FileRepo         SkillFileRepository
	TagRepo          SkillTagRepository
	VersionStatsRepo SkillVersionStatsRepository
	CompRepo         SkillStorageDeletionCompensationRepository

	// Infrastructure dependencies.
	Store        storage.Store
	Transactor   uow.Transactor

	// Optional validators (nil = defaults).
	PackageValidator  *packagekit.SkillPackageValidator
	MetadataParser    *packagekit.SkillMetadataParser
	PrePublishValidator PrePublishValidator
	Scanner           SecurityScanner
}

// NewService creates a fully wired skill Service.
func NewService(cfg ServiceConfig) *Service {
	validator := cfg.PackageValidator
	if validator == nil {
		validator = packagekit.NewSkillPackageValidator(cfg.MetadataParser)
	}
	parser := cfg.MetadataParser
	if parser == nil {
		parser = packagekit.NewSkillMetadataParser()
	}
	visibility := NewVisibilityChecker()
	installability := &SkillInstallability{}

	publish := NewSkillPublishService(
		cfg.NamespaceRepo,
		cfg.NamespaceMemberRepo,
		cfg.SkillRepo,
		cfg.VersionRepo,
		cfg.FileRepo,
		cfg.Store,
		validator,
		parser,
		cfg.PrePublishValidator,
		cfg.Scanner,
		cfg.Transactor,
	)

	download := NewSkillDownloadService(
		cfg.NamespaceRepo,
		cfg.SkillRepo,
		cfg.VersionRepo,
		cfg.FileRepo,
		cfg.TagRepo,
		cfg.Store,
		visibility,
	)

	query := NewSkillQueryService(
		cfg.NamespaceRepo,
		cfg.SkillRepo,
		cfg.VersionRepo,
		cfg.FileRepo,
		cfg.TagRepo,
		cfg.Store,
		visibility,
	)

	tags := NewSkillTagService(cfg.TagRepo, cfg.VersionRepo)

	deleteSvc := NewSkillHardDeleteService(
		cfg.SkillRepo,
		cfg.VersionRepo,
		cfg.FileRepo,
		cfg.TagRepo,
		cfg.VersionStatsRepo,
		cfg.CompRepo,
		cfg.Store,
	)

	return &Service{
		Publish:        publish,
		Download:       download,
		Query:          query,
		Tags:           tags,
		Delete:         deleteSvc,
		Visibility:     visibility,
		Installability: installability,
	}
}
