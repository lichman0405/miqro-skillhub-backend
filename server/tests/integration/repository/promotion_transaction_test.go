package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	postgresadapters "miqro-skillhub/server/internal/adapters/postgres"
	testutil "miqro-skillhub/server/internal/testutil/postgres"
	"miqro-skillhub/server/sdk/skillhub/auth"
	"miqro-skillhub/server/sdk/skillhub/eventbus"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/promotion"
	"miqro-skillhub/server/sdk/skillhub/review"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// errInjectedFileCopy is the deterministic error injected during file copy to
// prove transaction rollback.
var errInjectedFileCopy = errors.New("injected file copy failure")

// ============================================================================
// Failure-injecting file repository for rollback tests
// ============================================================================

// failingAfterFirstFileRepo wraps a real skill.SkillFileRepository and fails
// SaveAll after saving the first file through the inner repo.  This simulates
// a partial write inside a transaction so we can prove rollback.
type failingAfterFirstFileRepo struct {
	inner skill.SkillFileRepository
}

func (f *failingAfterFirstFileRepo) FindByVersionID(ctx context.Context, versionID int64) ([]skill.SkillFile, error) {
	return f.inner.FindByVersionID(ctx, versionID)
}

func (f *failingAfterFirstFileRepo) Save(ctx context.Context, file skill.SkillFile) (skill.SkillFile, error) {
	return f.inner.Save(ctx, file)
}

func (f *failingAfterFirstFileRepo) SaveAll(ctx context.Context, files []skill.SkillFile) ([]skill.SkillFile, error) {
	if len(files) == 0 {
		return nil, errInjectedFileCopy
	}
	// Save the first file through the real repository inside the same
	// transaction context.  The transaction will be rolled back when we
	// return the error.
	if _, err := f.inner.Save(ctx, files[0]); err != nil {
		return nil, err
	}
	return nil, errInjectedFileCopy
}

func (f *failingAfterFirstFileRepo) DeleteByVersionID(ctx context.Context, versionID int64) error {
	return f.inner.DeleteByVersionID(ctx, versionID)
}

// ============================================================================
// Recording notifier for side-effect assertions
// ============================================================================

type recorderNotifier struct {
	notified int
}

func (n *recorderNotifier) NotifyUser(_ context.Context, userID, category, entityType string, entityID int64, title, bodyJSON string) error {
	n.notified++
	return nil
}

// ============================================================================
// Fixture setup
// ============================================================================

// promotionFixture holds all the wired-up dependencies for a promotion
// integration test.
type promotionFixture struct {
	db            *postgresadapters.DB
	transactor    *postgresadapters.Transactor
	userRepo      *postgresadapters.UserAccountRepo
	namespaceRepo *postgresadapters.NamespaceRepo
	skillRepo     *postgresadapters.SkillRepo
	versionRepo   *postgresadapters.SkillVersionRepo
	fileRepo      *postgresadapters.SkillFileRepo
	promotionRepo *postgresadapters.PromotionRequestRepo
	eventBus      *eventbus.NoopBus
	notifier      *recorderNotifier
	svc           *promotion.PromotionService
	globalNSID    int64
	sourceNSID    int64
	sourceSkillID int64
	srcVersionID  int64
	promotionID   int64
	submitterID   string
	reviewerID    string
}

// setupPromotionFixture creates all necessary seed data and wires a real
// PromotionService backed by real PostgreSQL repositories.  The global
// namespace must already exist from migrations/seed data.
func setupPromotionFixture(t *testing.T) *promotionFixture {
	t.Helper()

	db := testutil.TestDB(t)
	ctx := context.Background()
	uid := fmt.Sprintf("%d", time.Now().UnixNano())

	f := &promotionFixture{
		db:            db,
		transactor:    postgresadapters.NewTransactor(db.Pool),
		userRepo:      postgresadapters.NewUserAccountRepo(db),
		namespaceRepo: postgresadapters.NewNamespaceRepo(db),
		skillRepo:     postgresadapters.NewSkillRepo(db),
		versionRepo:   postgresadapters.NewSkillVersionRepo(db),
		fileRepo:      postgresadapters.NewSkillFileRepo(db),
		promotionRepo: postgresadapters.NewPromotionRequestRepo(db),
		submitterID:   "submitter-" + uid,
		reviewerID:    "reviewer-" + uid,
	}

	// ── Users ──────────────────────────────────────────────────────────
	_, err := f.userRepo.Save(ctx, auth.UserAccount{
		ID: f.submitterID, DisplayName: "Submitter", Status: "ACTIVE",
	})
	if err != nil {
		t.Fatalf("create submitter: %v", err)
	}
	_, err = f.userRepo.Save(ctx, auth.UserAccount{
		ID: f.reviewerID, DisplayName: "Reviewer", Status: "ACTIVE",
	})
	if err != nil {
		t.Fatalf("create reviewer: %v", err)
	}

	// ── Namespaces ─────────────────────────────────────────────────────
	sourceNS, err := f.namespaceRepo.Save(ctx, namespace.Namespace{
		Slug: "source-team-" + uid, DisplayName: "Source Team",
		Type: "TEAM", Status: "ACTIVE",
	})
	if err != nil {
		t.Fatalf("create source namespace: %v", err)
	}
	f.sourceNSID = sourceNS.ID

	globalNS, err := f.namespaceRepo.FindBySlug(ctx, "global")
	if err != nil {
		t.Fatalf("find global namespace: %v", err)
	}
	f.globalNSID = globalNS.ID

	// ── Source skill + version + files ────────────────────────────────
	sourceSkill, err := f.skillRepo.Save(ctx, skill.Skill{
		NamespaceID: f.sourceNSID, Slug: "source-skill-" + uid,
		DisplayName: "Source Skill", Summary: "For integration test",
		OwnerID: f.submitterID, Visibility: "PUBLIC", Status: "ACTIVE",
	})
	if err != nil {
		t.Fatalf("create source skill: %v", err)
	}
	f.sourceSkillID = sourceSkill.ID

	sourceVersion, err := f.versionRepo.Save(ctx, skill.SkillVersion{
		SkillID: f.sourceSkillID, Version: "1.0.0", Status: "PUBLISHED",
		FileCount: 2, TotalSize: 200,
	})
	if err != nil {
		t.Fatalf("create source version: %v", err)
	}
	f.srcVersionID = sourceVersion.ID

	// Two files so partial copy failure can prove rollback of the first.
	_, err = f.fileRepo.Save(ctx, skill.SkillFile{
		VersionID: f.srcVersionID, FilePath: "SKILL.md", FileSize: 100,
		ContentType: "text/markdown", SHA256: "aaa111",
		StorageKey: "skills/" + uid + "/SKILL.md",
	})
	if err != nil {
		t.Fatalf("create source file 1: %v", err)
	}
	_, err = f.fileRepo.Save(ctx, skill.SkillFile{
		VersionID: f.srcVersionID, FilePath: "tools/main.py", FileSize: 100,
		ContentType: "text/x-python", SHA256: "bbb222",
		StorageKey: "skills/" + uid + "/main.py",
	})
	if err != nil {
		t.Fatalf("create source file 2: %v", err)
	}

	// ── Promotion request ─────────────────────────────────────────────
	savedReq, err := f.promotionRepo.Save(ctx, review.PromotionRequest{
		SourceSkillID:     f.sourceSkillID,
		SourceVersionID:   f.srcVersionID,
		TargetNamespaceID: f.globalNSID,
		SubmittedBy:       f.submitterID,
		Status:            string(review.ReviewStatusPending),
		Version:           1,
	})
	if err != nil {
		t.Fatalf("create promotion request: %v", err)
	}
	f.promotionID = savedReq.ID

	// ── Service ───────────────────────────────────────────────────────
	f.eventBus = eventbus.NewNoopBus(true)
	f.notifier = &recorderNotifier{}
	f.svc = promotion.NewPromotionService(
		f.promotionRepo, f.skillRepo, f.versionRepo, f.fileRepo,
		f.namespaceRepo, nil, f.eventBus, f.notifier,
	)
	f.svc.SetTransactor(f.transactor)

	return f
}

// countRows runs a count query and returns the result.
func countRows(t *testing.T, db *postgresadapters.DB, query string, args ...any) int {
	t.Helper()
	var count int
	if err := db.Pool.QueryRow(context.Background(), query, args...).Scan(&count); err != nil {
		t.Fatalf("countRows %q: %v", query, err)
	}
	return count
}

// ============================================================================
// Integration tests
// ============================================================================

func TestPromotionApprove_PostgresTransaction_CommitsAllWrites(t *testing.T) {
	f := setupPromotionFixture(t)
	ctx := context.Background()

	approved, err := f.svc.ApprovePromotion(ctx, f.promotionID, f.reviewerID,
		"looks good", map[string]bool{"SKILL_ADMIN": true})
	if err != nil {
		t.Fatalf("ApprovePromotion failed: %v", err)
	}

	// ── Returned value assertions ─────────────────────────────────────
	if approved.Status != string(review.ReviewStatusApproved) {
		t.Errorf("expected APPROVED, got %s", approved.Status)
	}
	if approved.TargetSkillID == nil || *approved.TargetSkillID == 0 {
		t.Fatal("expected target_skill_id to be populated")
	}
	targetSkillID := *approved.TargetSkillID

	// ── promotion_request row committed ───────────────────────────────
	status, tsID, comment := "", (*int64)(nil), (*string)(nil)
	err = f.db.Pool.QueryRow(ctx,
		`SELECT status, target_skill_id, review_comment FROM promotion_request WHERE id = $1`,
		f.promotionID,
	).Scan(&status, &tsID, &comment)
	if err != nil {
		t.Fatalf("re-fetch promotion request: %v", err)
	}
	if status != string(review.ReviewStatusApproved) {
		t.Errorf("promotion_request.status = %s, want APPROVED", status)
	}
	if tsID == nil || *tsID == 0 {
		t.Error("promotion_request.target_skill_id should be non-null")
	}
	if comment == nil || *comment != "looks good" {
		t.Errorf("promotion_request.review_comment = %v, want 'looks good'", comment)
	}

	// ── Target skill in global namespace ──────────────────────────────
	var skillExists bool
	err = f.db.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM skill WHERE id = $1 AND namespace_id = $2)`,
		targetSkillID, f.globalNSID,
	).Scan(&skillExists)
	if err != nil {
		t.Fatalf("check target skill: %v", err)
	}
	if !skillExists {
		t.Fatal("target skill does not exist in global namespace")
	}

	var sourceSkillID int64
	err = f.db.Pool.QueryRow(ctx,
		`SELECT source_skill_id FROM skill WHERE id = $1`, targetSkillID,
	).Scan(&sourceSkillID)
	if err != nil {
		t.Fatalf("read target skill source_skill_id: %v", err)
	}
	if sourceSkillID != f.sourceSkillID {
		t.Errorf("target skill source_skill_id = %d, want %d", sourceSkillID, f.sourceSkillID)
	}

	// ── Target version linked via latest_version_id ───────────────────
	var latestVersionID int64
	err = f.db.Pool.QueryRow(ctx,
		`SELECT latest_version_id FROM skill WHERE id = $1`, targetSkillID,
	).Scan(&latestVersionID)
	if err != nil {
		t.Fatalf("read latest_version_id: %v", err)
	}
	if latestVersionID == 0 {
		t.Fatal("target skill has no latest_version_id")
	}

	var verStatus string
	err = f.db.Pool.QueryRow(ctx,
		`SELECT status FROM skill_version WHERE id = $1`, latestVersionID,
	).Scan(&verStatus)
	if err != nil {
		t.Fatalf("read target version status: %v", err)
	}
	if verStatus != "PUBLISHED" {
		t.Errorf("target version status = %s, want PUBLISHED", verStatus)
	}

	// ── Target files copied ───────────────────────────────────────────
	fileCount := countRows(t, f.db,
		`SELECT COUNT(*) FROM skill_file WHERE version_id = $1`, latestVersionID)
	if fileCount != 2 {
		t.Errorf("target file count = %d, want 2", fileCount)
	}

	// ── Post-commit events emitted ────────────────────────────────────
	if len(f.eventBus.Events) < 2 {
		t.Errorf("expected at least 2 events, got %d", len(f.eventBus.Events))
	}
	if f.notifier.notified == 0 {
		t.Error("expected notification after successful approval")
	}
}

func TestPromotionApprove_PostgresTransaction_RollsBackPartialWritesOnFileCopyFailure(t *testing.T) {
	f := setupPromotionFixture(t)
	ctx := context.Background()

	// Wrap the file repo to fail after saving the first file.
	wrapped := &failingAfterFirstFileRepo{inner: f.fileRepo}

	// Build a fresh service with the wrapped file repo but real everything else.
	notifier := &recorderNotifier{}
	bus := eventbus.NewNoopBus(true)
	svc2 := promotion.NewPromotionService(
		f.promotionRepo, f.skillRepo, f.versionRepo, wrapped,
		f.namespaceRepo, nil, bus, notifier,
	)
	svc2.SetTransactor(f.transactor)

	_, err := svc2.ApprovePromotion(ctx, f.promotionID, f.reviewerID,
		"looks good", map[string]bool{"SKILL_ADMIN": true})
	if err == nil {
		t.Fatal("expected error from injected file copy failure")
	}
	if !errors.Is(err, errInjectedFileCopy) {
		t.Logf("error: %v", err)
	}

	// ── promotion_request rolled back ─────────────────────────────────
	status, tsID := "", (*int64)(nil)
	err = f.db.Pool.QueryRow(ctx,
		`SELECT status, target_skill_id FROM promotion_request WHERE id = $1`,
		f.promotionID,
	).Scan(&status, &tsID)
	if err != nil {
		t.Fatalf("re-fetch promotion request: %v", err)
	}
	if status != string(review.ReviewStatusPending) {
		t.Errorf("promotion_request.status = %s, want PENDING (rolled back)", status)
	}
	if tsID != nil {
		t.Errorf("promotion_request.target_skill_id = %v, want nil (rolled back)", tsID)
	}

	// ── No target skill in global namespace ───────────────────────────
	skillsInGlobal := countRows(t, f.db,
		`SELECT COUNT(*) FROM skill WHERE namespace_id = $1 AND slug LIKE 'source-skill-%'`,
		f.globalNSID)
	if skillsInGlobal > 0 {
		t.Errorf("found %d skill(s) in global namespace after rollback, want 0", skillsInGlobal)
	}

	// ── No residual target files ──────────────────────────────────────
	// All files should still belong only to the source version.
	totalFiles := countRows(t, f.db, `SELECT COUNT(*) FROM skill_file`)
	if totalFiles != 2 {
		t.Errorf("total file count = %d, want 2 (only source files)", totalFiles)
	}

	// ── No post-commit side effects ───────────────────────────────────
	if len(bus.Events) > 0 {
		t.Errorf("expected 0 events after rollback, got %d", len(bus.Events))
	}
	if notifier.notified > 0 {
		t.Errorf("expected 0 notifications after rollback, got %d", notifier.notified)
	}
}
