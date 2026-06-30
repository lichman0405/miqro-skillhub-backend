package social_test

import (
	"context"
	"fmt"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/eventbus"
	"miqro-skillhub/server/sdk/skillhub/social"
)

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
func (m *mockStarRepo) DeleteBySkillID(_ context.Context, skillID int64) error { return nil }
func (m *mockStarRepo) CountBySkillID(_ context.Context, skillID int64) (int64, error) { return 0, nil }

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

	// Idempotent: star again does not error.
	err = svc.Star(context.Background(), 10, "user-1")
	if err != nil {
		t.Fatalf("second Star failed: %v", err)
	}

	// Unstar.
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

	// Unstar a non-existent star is idempotent.
	err := svc.Unstar(context.Background(), 10, "user-1")
	if err != nil {
		t.Fatalf("Unstar nonexistent should not error: %v", err)
	}
}
