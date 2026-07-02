// Command skillhub-server is the main HTTP server process for SkillHub.
//
// It loads configuration from the environment, wires adapters, and
// starts the HTTP listener.  All core behavior lives in the SDK
// packages under server/sdk/skillhub; this binary is the process
// adapter.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"miqro-skillhub/server/internal/adapters/agentrunner"
	"miqro-skillhub/server/internal/adapters/localstorage"
	"miqro-skillhub/server/internal/adapters/postgres"
	"miqro-skillhub/server/internal/config"
	httpx "miqro-skillhub/server/internal/http"
	"miqro-skillhub/server/internal/http/cliapi"
	"miqro-skillhub/server/internal/http/frontend"
	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/observability"
	"miqro-skillhub/server/internal/http/portal"
	"miqro-skillhub/server/internal/http/toolapi"
	"miqro-skillhub/server/sdk/skillhub/agentci"
	"miqro-skillhub/server/sdk/skillhub/audit"
	"miqro-skillhub/server/sdk/skillhub/auth"
	"miqro-skillhub/server/sdk/skillhub/community"
	"miqro-skillhub/server/sdk/skillhub/eventbus"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/packagekit"
	"miqro-skillhub/server/sdk/skillhub/release"
	"miqro-skillhub/server/sdk/skillhub/review"
	"miqro-skillhub/server/sdk/skillhub/search"
	"miqro-skillhub/server/sdk/skillhub/skill"
	"miqro-skillhub/server/sdk/skillhub/storage"
	"miqro-skillhub/server/sdk/skillhub/tooling"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// ── Database ──────────────────────────────────────────────────────────
	ctx := context.Background()
	var db *postgres.DB
	db, err = postgres.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("WARNING: cannot connect to database: %v — routes will return 503", err)
	}

	// ── SDK services ──────────────────────────────────────────────────────
	var (
		authSvc        *auth.Service
		nsSvc          *namespace.Service
		skillSvc       *skill.Service
		srcSvc         *search.Service
		releaseSvc     *release.Service
		communitySvc   *community.Service
		agentciSvc     *agentci.Service
		reviewSvc      *review.ReviewService
		objStore       storage.Store
		limiter        *middleware.RateLimiter
		authMW         *middleware.AuthMiddleware
		validator      *packagekit.SkillPackageValidator
		metadataParser *packagekit.SkillMetadataParser
	)

	if db != nil {
		// Repositories.
		userRepo := postgres.NewUserAccountRepo(db)
		localCredRepo := postgres.NewLocalCredentialRepo(db)
		tokenRepo := postgres.NewApiTokenRepo(db)
		roleRepo := postgres.NewRoleRepo(db)
		permRepo := postgres.NewPermissionRepo(db)
		roleBindingRepo := postgres.NewUserRoleBindingRepo(db)
		identityBindingRepo := postgres.NewIdentityBindingRepo(db)
		pwdResetRepo := postgres.NewPasswordResetRequestRepo(db)
		mergeRepo := postgres.NewAccountMergeRequestRepo(db)
		nsRepo := postgres.NewNamespaceRepo(db)
		nsMemberRepo := postgres.NewNamespaceMemberRepo(db)
		skillRepo := postgres.NewSkillRepo(db)
		versionRepo := postgres.NewSkillVersionRepo(db)
		fileRepo := postgres.NewSkillFileRepo(db)
		tagRepo := postgres.NewSkillTagRepo(db)
		searchQueryRepo := postgres.NewSearchQueryRepo(db)

		// Auth service.
		authSvc = auth.NewService(auth.ServiceConfig{
			UserAccountRepo:     userRepo,
			LocalCredentialRepo: localCredRepo,
			ApiTokenRepo:        tokenRepo,
			RoleRepo:            roleRepo,
			PermissionRepo:      permRepo,
			UserRoleBindingRepo: roleBindingRepo,
			IdentityBindingRepo: identityBindingRepo,
			PasswordResetRepo:   pwdResetRepo,
			AccountMergeRepo:    mergeRepo,
		})

		// Namespace service.
		nsSvc = namespace.NewService(namespace.ServiceConfig{
			NamespaceRepo: nsRepo,
			MemberRepo:    nsMemberRepo,
		})

		// Skill service.
		metadataParser = packagekit.NewSkillMetadataParser()
		validator = packagekit.NewSkillPackageValidator(metadataParser)

		// Object storage for development — local filesystem.
		storageRoot := os.Getenv("STORAGE_ROOT")
		if storageRoot == "" {
			storageRoot = "./data/storage"
		}
		objStore, err = localstorage.New(storageRoot)
		if err != nil {
			log.Fatalf("object storage: %v", err)
		}

		skillSvc = skill.NewService(skill.ServiceConfig{
			NamespaceRepo:       nsRepo,
			NamespaceMemberRepo: nsMemberRepo,
			SkillRepo:           skillRepo,
			VersionRepo:         versionRepo,
			FileRepo:            fileRepo,
			TagRepo:             tagRepo,
			Store:               objStore,
			MetadataParser:      metadataParser,
			PackageValidator:    validator,
		})

		// Search service.
		srcSvc = &search.Service{
			Query: searchQueryRepo,
		}

		// Release service.
		{
			releaseRepo := postgres.NewReleaseRepo(db)
			releaseAssetRepo := postgres.NewReleaseAssetRepo(db)
			releaseSvc = release.NewService(releaseRepo, releaseAssetRepo, versionRepo)
		}

		// Community service.
		{
			communitySvc = community.NewService(
				postgres.NewIssueRepo(db),
				postgres.NewIssueCommentRepo(db),
				postgres.NewDiscussionRepo(db),
				postgres.NewDiscCommentRepo(db),
				postgres.NewWikiPageRepo(db),
				postgres.NewWikiVersionRepo(db),
				postgres.NewChangeProposalRepo(db),
				postgres.NewProposalCommentRepo(db),
				postgres.NewIssueLabelRepo(db),
				postgres.NewDiscussionLabelRepo(db),
				postgres.NewCommunityReportRepo(db),
			)
		}

		// Agent CI service.
		{
			agentciSvc = agentci.NewService(
				postgres.NewPipelineDefinitionRepo(db),
				postgres.NewPipelineRunRepo(db),
				postgres.NewCheckRunRepo(db),
				postgres.NewCheckStepRepo(db),
				postgres.NewCheckArtifactRepo(db),
				postgres.NewGatePolicyRepo(db),
				postgres.NewAgentWorkerRepo(db),
				nil, // log store not yet wired
			)

			// Register local deterministic runner with version file reader.
			localRunner := agentrunner.NewLocalRunner()
			localRunner.SetVersionFileReader(func(ctx context.Context, versionID, skillID int64) ([]agentci.PackageFileEntry, error) {
				files, err := fileRepo.FindByVersionID(ctx, versionID)
				if err != nil {
					return nil, fmt.Errorf("find files: %w", err)
				}
				entries := make([]agentci.PackageFileEntry, 0, len(files))
				for _, f := range files {
					var content []byte
					if f.StorageKey != "" && objStore != nil {
						rc, getErr := objStore.GetObject(ctx, f.StorageKey)
						if getErr == nil {
							content, _ = io.ReadAll(rc)
							rc.Close()
						}
					}
					entries = append(entries, agentci.PackageFileEntry{
						Path:        f.FilePath,
						Content:     content,
						Size:        f.FileSize,
						ContentType: f.ContentType,
					})
				}
				return entries, nil
			})
			agentciSvc.RegisterRunner(localRunner)
		}

		// Review service — wired with CI gate enforcement for review approval.
		{
			reviewTaskRepo := postgres.NewReviewTaskRepo(db)
			reviewSvc = review.NewReviewService(
				reviewTaskRepo,
				versionRepo,
				skillRepo,
				nsRepo,
				nil, // permission checker (default)
				eventbus.NewNoopBus(true),
				nil, // notifier
			)
			// Wire gate enforcement — SDK-first, not just HTTP handler.
			if agentciSvc != nil {
				reviewSvc.SetGateEnforcer(func(ctx context.Context, skillID, versionID int64, triggerType string) error {
					return agentciSvc.GateEnforce(ctx, agentci.GateEvalRequest{
						SkillID:     skillID,
						VersionID:   &versionID,
						TriggerType: triggerType,
					})
				})
			}
		}

		// Auth middleware with full namespace projection.
		authMW = middleware.NewAuthMiddleware(
			nil,           // session store (not wired yet)
			authSvc.Token, // bearer token validation
			authSvc.RBAC,  // platform role lookup
			userRepo,      // user profile lookup
			nsMemberRepo,  // namespace membership projection
		)
	}

	// Rate limiter — always available.
	limiter = middleware.NewRateLimiter(100, 10.0)

	// ── HTTP route groups ─────────────────────────────────────────────────
	var (
		handlerAuth      *portal.AuthHandler
		handlerNamespace *portal.NamespaceHandler
		handlerSkill     *portal.SkillHandler
		handlerSearch    *portal.SearchHandler
		handlerCLI       *cliapi.Handler
	)

	if authSvc != nil && skillSvc != nil && nsSvc != nil && srcSvc != nil {
		handlerAuth = &portal.AuthHandler{AuthSvc: authSvc}
		handlerNamespace = &portal.NamespaceHandler{NsSvc: nsSvc}
		handlerSkill = &portal.SkillHandler{
			SkillSvc:         skillSvc,
			PackageValidator: validator,
			MetadataParser:   metadataParser,
		}
		handlerSearch = &portal.SearchHandler{SearchSvc: srcSvc}
		handlerCLI = &cliapi.Handler{SkillSvc: skillSvc, SearchSvc: srcSvc}
	}

	// Release handler — always constructed when the release service is available.
	var handlerRelease *portal.ReleaseHandler
	if releaseSvc != nil && skillSvc != nil {
		handlerRelease = &portal.ReleaseHandler{ReleaseSvc: releaseSvc, SkillSvc: skillSvc, AgentCISvc: agentciSvc}
	}

	// Community handler — always constructed when the community service is available.
	var handlerCommunity *portal.CommunityHandler
	var frontendCommunity *frontend.CommunityFrontendHandler
	if communitySvc != nil && skillSvc != nil && db != nil {
		// Wire version/release lookups for cross-skill validation.
		communitySvc.SetVersionLookup(postgres.NewCommunityVersionLookup(db))
		communitySvc.SetReleaseLookup(postgres.NewCommunityReleaseLookup(db))

		// Wire event publisher via event bus.
		bus := eventbus.NewNoopBus(true)
		communitySvc.SetEventPublisher(postgres.NewCommunityEventPublisher(bus))

		// Wire audit recorder.
		auditRepo := postgres.NewAuditLogRepo(db)
		auditSvc := audit.NewAuditLogService(auditRepo)
		communitySvc.SetAuditRecorder(postgres.NewCommunityAuditRecorder(auditSvc))

		// Wire community search repository.
		communitySvc.SetSearchRepo(postgres.NewCommunitySearchRepo(db))

		handlerCommunity = &portal.CommunityHandler{CommunitySvc: communitySvc, SkillSvc: skillSvc}
		frontendCommunity = &frontend.CommunityFrontendHandler{CommunitySvc: communitySvc, SkillH: handlerSkill}
	}

	// Agent CI handler — always constructed when agent CI service is available.
	var handlerAgentCI *portal.AgentCIHandler
	if agentciSvc != nil && skillSvc != nil {
		handlerAgentCI = &portal.AgentCIHandler{AgentCISvc: agentciSvc, SkillSvc: skillSvc}
	}

	// Tool API handler — always constructed when the skill service is available.
	var handlerToolAPI *toolapi.Handler
	if skillSvc != nil {
		toolingSvc := tooling.NewService(skillSvc)
		if agentciSvc != nil {
			toolingSvc.SetAgentCIService(agentciSvc)
		}
		handlerToolAPI = &toolapi.Handler{Tooling: toolingSvc}
	}

	// ── Router ────────────────────────────────────────────────────────────
	metricsReg := observability.NewMetricsRegistry()
	router := httpx.NewRouter(httpx.RouterConfig{
		Health:            &httpx.HealthHandler{},
		AuthMW:            authMW,
		RateLimiter:       limiter,
		PortalAuth:        handlerAuth,
		PortalNamespace:   handlerNamespace,
		PortalSkill:       handlerSkill,
		PortalSearch:      handlerSearch,
		PortalRelease:     handlerRelease,
		PortalCommunity:   handlerCommunity,
		PortalAgentCI:     handlerAgentCI,
		FrontendCommunity: frontendCommunity,
		CLI:               handlerCLI,
		ToolAPI:           handlerToolAPI,
		MetricsRegistry:   metricsReg,
	})

	// Wrap with structured request logging, metrics instrumentation, and optional browser CORS.
	var handler http.Handler = observability.NewRequestLogger(router, log.Default(), metricsReg)
	handler = middleware.NewCORSMiddleware(cfg.CORSAllowedOrigins).Wrap(handler)

	srv := &http.Server{
		Addr:         cfg.APIAddr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("skillhub-server listening on %s", cfg.APIAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	if db != nil {
		db.Close()
	}
	log.Println("skillhub-server stopped")
}
