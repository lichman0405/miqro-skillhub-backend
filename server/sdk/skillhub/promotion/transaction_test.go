package promotion_test

import (
	"context"
	"errors"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/promotion"
	"miqro-skillhub/server/sdk/skillhub/review"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// ============================================================================
// Fake transactor for transaction-boundary tests
// ============================================================================

type recordingTransactor struct {
	called   bool
	rollback bool
	// when non-nil, the fn passed to WithinTx fails with this error.
	injectFnError error
}

func (t *recordingTransactor) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	t.called = true
	err := fn(ctx)
	if t.injectFnError != nil {
		return t.injectFnError
	}
	if err != nil {
		t.rollback = true
	}
	return err
}

// ============================================================================
// Notifier that records calls for transaction tests
// ============================================================================

type recordingNotifier struct{ notified int }

func (n *recordingNotifier) NotifyUser(_ context.Context, userID, category, entityType string, entityID int64, title, bodyJSON string) error {
	n.notified++
	return nil
}

// ============================================================================
// File repo that can be configured to fail on SaveAll
// ============================================================================

type fileRepoWithFailure struct {
	*mockSkillFileRepo
	failOnSaveAll error
}

func newFileRepoWithFailure(base *mockSkillFileRepo) *fileRepoWithFailure {
	return &fileRepoWithFailure{mockSkillFileRepo: base}
}

func (m *fileRepoWithFailure) SaveAll(ctx context.Context, files []skill.SkillFile) ([]skill.SkillFile, error) {
	if m.failOnSaveAll != nil {
		return nil, m.failOnSaveAll
	}
	return m.mockSkillFileRepo.SaveAll(ctx, files)
}

// ============================================================================
// Transaction-boundary tests
// ============================================================================

func TestApprovePromotion_UsesTransactorForAllWrites(t *testing.T) {
	_, skillRepo, _, _, _, svc := setupPromotionService()
	req, _ := svc.SubmitPromotion(context.Background(), 10, 20, 2, "owner-1", ownerRoles(), nil)

	rec := &recordingTransactor{}
	svc.SetTransactor(rec)

	approved, err := svc.ApprovePromotion(context.Background(), req.ID, "plat-admin", "", platformAdmin())
	if err != nil {
		t.Fatalf("ApprovePromotion failed: %v", err)
	}
	if approved.Status != string(review.ReviewStatusApproved) {
		t.Errorf("expected APPROVED, got %s", approved.Status)
	}
	if !rec.called {
		t.Fatal("expected transactor to be called")
	}
	if rec.rollback {
		t.Fatal("expected no rollback")
	}

	// Verify data was committed through transactor context.
	skills, _ := skillRepo.FindByNamespaceIDAndSlug(context.Background(), 2, "my-skill")
	if len(skills) == 0 {
		t.Fatal("expected new skill in global namespace after transactor commit")
	}
}

func TestApprovePromotion_SaveFilesFailureRollsBackAndDoesNotNotify(t *testing.T) {
	reqRepo, skillRepo, verRepo, fileRepo, nsRepo, svc := setupPromotionService()
	req, _ := svc.SubmitPromotion(context.Background(), 10, 20, 2, "owner-1", ownerRoles(), nil)

	// Wrap the file repo with failure injection.
	failingFileRepo := newFileRepoWithFailure(fileRepo)
	failingFileRepo.failOnSaveAll = errors.New("simulated disk error")

	notifier := &recordingNotifier{}

	svc2 := promotion.NewPromotionService(
		reqRepo, skillRepo, verRepo, failingFileRepo, nsRepo,
		nil, nil, notifier,
	)

	rec := &recordingTransactor{}
	svc2.SetTransactor(rec)

	_, err := svc2.ApprovePromotion(context.Background(), req.ID, "plat-admin", "", platformAdmin())
	if err == nil {
		t.Fatal("expected error from SaveAll failure")
	}
	if !rec.called {
		t.Fatal("expected transactor to be called")
	}
	// Notification must NOT be sent when transaction fails.
	if notifier.notified > 0 {
		t.Fatalf("notification should not be sent when transaction fails, got %d notifications", notifier.notified)
	}
}

func TestApprovePromotion_NoTransactorStillWorks(t *testing.T) {
	_, _, _, _, _, svc := setupPromotionService()
	req, _ := svc.SubmitPromotion(context.Background(), 10, 20, 2, "owner-1", ownerRoles(), nil)

	// No transactor set — should work like before.
	approved, err := svc.ApprovePromotion(context.Background(), req.ID, "plat-admin", "", platformAdmin())
	if err != nil {
		t.Fatalf("ApprovePromotion without transactor failed: %v", err)
	}
	if approved.Status != string(review.ReviewStatusApproved) {
		t.Errorf("expected APPROVED, got %s", approved.Status)
	}
}
