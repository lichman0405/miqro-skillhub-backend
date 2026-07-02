// Command skillhub-server is the main HTTP server process for SkillHub.
//
// It loads configuration from the environment, wires adapters, and
// starts the HTTP listener.  All core behavior lives in the SDK
// packages under server/sdk/skillhub; this binary is the process
// adapter.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"miqro-skillhub/server/internal/adapters/postgres"
	"miqro-skillhub/server/internal/config"
	httpx "miqro-skillhub/server/internal/http"
	"miqro-skillhub/server/internal/http/cliapi"
	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/observability"
	"miqro-skillhub/server/internal/http/portal"
	"miqro-skillhub/server/internal/http/toolapi"
	"miqro-skillhub/server/sdk/skillhub/auth"
	"miqro-skillhub/server/sdk/skillhub/community"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/packagekit"
	"miqro-skillhub/server/sdk/skillhub/release"
	"miqro-skillhub/server/sdk/skillhub/search"
	"miqro-skillhub/server/sdk/skillhub/skill"
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
		skillSvc = skill.NewService(skill.ServiceConfig{
			NamespaceRepo:       nsRepo,
			NamespaceMemberRepo: nsMemberRepo,
			SkillRepo:           skillRepo,
			VersionRepo:         versionRepo,
			FileRepo:            fileRepo,
			TagRepo:             tagRepo,
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

		// Auth middleware with full namespace projection.
		authMW = middleware.NewAuthMiddleware(
			nil,            // session store (not wired yet)
			authSvc.Token,  // bearer token validation
			authSvc.RBAC,   // platform role lookup
			userRepo,       // user profile lookup
			nsMemberRepo,   // namespace membership projection
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
		handlerRelease = &portal.ReleaseHandler{ReleaseSvc: releaseSvc, SkillSvc: skillSvc}
	}

	// Community handler — always constructed when the community service is available.
	var handlerCommunity *portal.CommunityHandler
	if communitySvc != nil && skillSvc != nil {
		handlerCommunity = &portal.CommunityHandler{CommunitySvc: communitySvc, SkillSvc: skillSvc}
	}

	// Tool API handler — always constructed when the skill service is available.
	var handlerToolAPI *toolapi.Handler
	if skillSvc != nil {
		toolingSvc := tooling.NewService(skillSvc)
		handlerToolAPI = &toolapi.Handler{Tooling: toolingSvc}
	}

	// ── Router ────────────────────────────────────────────────────────────
	metricsReg := observability.NewMetricsRegistry()
	router := httpx.NewRouter(httpx.RouterConfig{
		Health:          &httpx.HealthHandler{},
		AuthMW:          authMW,
		RateLimiter:     limiter,
		PortalAuth:      handlerAuth,
		PortalNamespace: handlerNamespace,
		PortalSkill:     handlerSkill,
		PortalSearch:    handlerSearch,
		PortalRelease:   handlerRelease,
		PortalCommunity: handlerCommunity,
		CLI:             handlerCLI,
		ToolAPI:         handlerToolAPI,
		MetricsRegistry: metricsReg,
	})

	// Wrap with structured request logging + metrics instrumentation.
	rl := observability.NewRequestLogger(router, log.Default(), metricsReg)

	srv := &http.Server{
		Addr:         cfg.APIAddr,
		Handler:      rl,
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
