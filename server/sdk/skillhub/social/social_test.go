package social_test

import (
	"context"
	"fmt"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/eventbus"
	"miqro-skillhub/server/sdk/skillhub/social"
)

// ---- star mocks ----

type mockStarRepo struct {
	stars map[string]social.SkillStar
}

func newMockStarRepo() *mockStarRepo { return &mockStarRepo{stars: make(map[string]social.SkillStar)} }
func (m *mockStarRepo) Save(_ context.Context, s social.SkillStar) (social.SkillStar, error) {
	key := fmt.Sprintf("%d:%s", s.SkillID, s.UserID)
	if s.ID == 0 {
		s.ID = int64(len(m.stars) + 1)
	}
	m.stars[key] = s
	return s, nil
}
func (m *mockStarRepo) FindBySkillAndUser(_ context.Context, skillID int64, userID string) (*social.SkillStar, error) {
	key := fmt.Sprintf("%d:%s", skillID, userID)
	if s, ok := m.stars[key]; ok {
		return &s, nil
	}
	return nil, nil
}
func (m *mockStarRepo) Delete(_ context.Context, id int64) error {
	for k, s := range m.stars {
		if s.ID == id {
			delete(m.stars, k)
			return nil
		}
	}
	return nil
}
func (m *mockStarRepo) DeleteBySkillID(_ context.Context, _ int64) error { return nil }
func (m *mockStarRepo) CountBySkillID(_ context.Context, _ int64) (int64, error) { return 0, nil }

type alwaysExistsChecker struct{}

func (c *alwaysExistsChecker) Exists(_ context.Context, _ int64) (bool, error) { return true, nil }

func TestSocial_StarUnstar(t *testing.T) {
	starRepo := newMockStarRepo()
	svc := social.NewSkillStarService(starRepo, nil, &alwaysExistsChecker{}, eventbus.NewNoopBus(true))

	err := svc.Star(context.Background(), 10, "user-1")
	if err != nil {
		t.Fatalf("Star failed: %v", err)
	}

	starred, _ := svc.IsStarred(context.Background(), 10, "user-1")
	if !starred {
		t.Fatal("expected starred")
	}

	err = svc.Star(context.Background(), 10, "user-1")
	if err != nil {
		t.Fatalf("second Star failed: %v", err)
	}

	err = svc.Unstar(context.Background(), 10, "user-1")
	if err != nil {
		t.Fatalf("Unstar failed: %v", err)
	}

	starred, _ = svc.IsStarred(context.Background(), 10, "user-1")
	if starred {
		t.Fatal("expected unstarred")
	}
}

func TestSocial_Unstar_Nonexistent(t *testing.T) {
	starRepo := newMockStarRepo()
	svc := social.NewSkillStarService(starRepo, nil, &alwaysExistsChecker{}, eventbus.NewNoopBus(true))

	err := svc.Unstar(context.Background(), 10, "user-1")
	if err != nil {
		t.Fatalf("Unstar nonexistent should not error: %v", err)
	}
}

// ---- rating mocks ----

type mockRatingRepo struct {
	ratings map[string]social.SkillRating
}

func newMockRatingRepo() *mockRatingRepo {
	return &mockRatingRepo{ratings: make(map[string]social.SkillRating)}
}
func (m *mockRatingRepo) Save(_ context.Context, r social.SkillRating) (social.SkillRating, error) {
	key := fmt.Sprintf("%d:%s", r.SkillID, r.UserID)
	if r.ID == 0 {
		r.ID = int64(len(m.ratings) + 1)
	}
	m.ratings[key] = r
	return r, nil
}
func (m *mockRatingRepo) FindBySkillAndUser(_ context.Context, skillID int64, userID string) (*social.SkillRating, error) {
	key := fmt.Sprintf("%d:%s", skillID, userID)
	if r, ok := m.ratings[key]; ok {
		return &r, nil
	}
	return nil, nil
}
func (m *mockRatingRepo) AverageScoreBySkillID(_ context.Context, _ int64) (float64, error) {
	return 0, nil
}
func (m *mockRatingRepo) CountBySkillID(_ context.Context, _ int64) (int, error) { return 0, nil }
func (m *mockRatingRepo) DeleteBySkillID(_ context.Context, _ int64) error { return nil }

type mockRatingCounterUpdater struct {
	updateCalls []int64
}

func (c *mockRatingCounterUpdater) UpdateRatingStats(_ context.Context, skillID int64) error {
	c.updateCalls = append(c.updateCalls, skillID)
	return nil
}

func TestSocial_Rate_FirstRating(t *testing.T) {
	ratingRepo := newMockRatingRepo()
	counterUpdater := &mockRatingCounterUpdater{}
	svc := social.NewSkillRatingService(ratingRepo, &alwaysExistsChecker{}, counterUpdater, eventbus.NewNoopBus(true))

	err := svc.Rate(context.Background(), 10, "user-1", 4)
	if err != nil {
		t.Fatalf("Rate failed: %v", err)
	}

	got, _ := svc.GetUserRating(context.Background(), 10, "user-1")
	if got == nil || *got != 4 {
		t.Errorf("expected rating 4, got %v", got)
	}

	// Verify counter was updated.
	if len(counterUpdater.updateCalls) != 1 {
		t.Fatalf("expected 1 counter update call, got %d", len(counterUpdater.updateCalls))
	}
	if counterUpdater.updateCalls[0] != 10 {
		t.Errorf("expected counter update for skill 10, got %d", counterUpdater.updateCalls[0])
	}
}

func TestSocial_Rate_UpdateRating(t *testing.T) {
	ratingRepo := newMockRatingRepo()
	counterUpdater := &mockRatingCounterUpdater{}
	svc := social.NewSkillRatingService(ratingRepo, &alwaysExistsChecker{}, counterUpdater, eventbus.NewNoopBus(true))

	// First rating.
	svc.Rate(context.Background(), 10, "user-1", 3)

	// Update rating (count stays 1, avg changes).
	err := svc.Rate(context.Background(), 10, "user-1", 5)
	if err != nil {
		t.Fatalf("Rate update failed: %v", err)
	}

	got, _ := svc.GetUserRating(context.Background(), 10, "user-1")
	if got == nil || *got != 5 {
		t.Errorf("expected updated rating 5, got %v", got)
	}

	// Counter should be updated twice (once per Rate call).
	if len(counterUpdater.updateCalls) != 2 {
		t.Fatalf("expected 2 counter update calls, got %d", len(counterUpdater.updateCalls))
	}
}

func TestSocial_Rate_InvalidScore(t *testing.T) {
	ratingRepo := newMockRatingRepo()
	svc := social.NewSkillRatingService(ratingRepo, &alwaysExistsChecker{}, nil, eventbus.NewNoopBus(true))

	err := svc.Rate(context.Background(), 10, "user-1", 0)
	if err == nil {
		t.Fatal("expected error for score 0")
	}

	err = svc.Rate(context.Background(), 10, "user-1", 6)
	if err == nil {
		t.Fatal("expected error for score 6")
	}
}

func TestSocial_Rate_EventPublished(t *testing.T) {
	ratingRepo := newMockRatingRepo()
	counterUpdater := &mockRatingCounterUpdater{}
	bus := eventbus.NewNoopBus(true)
	svc := social.NewSkillRatingService(ratingRepo, &alwaysExistsChecker{}, counterUpdater, bus)

	svc.Rate(context.Background(), 10, "user-1", 5)

	if len(bus.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(bus.Events))
	}
	if bus.Events[0].EventName() != "social.rated" {
		t.Errorf("expected social.rated event, got %s", bus.Events[0].EventName())
	}
}

var _ = fmt.Sprintf
