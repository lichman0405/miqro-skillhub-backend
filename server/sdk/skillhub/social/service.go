package social

import (
	"context"
	"fmt"

	"miqro-skillhub/server/sdk/skillhub/eventbus"
)

// Service is the public facade for social interactions.
type Service struct {
	Stars         *SkillStarService
	Ratings       *SkillRatingService
	Subscriptions *SkillSubscriptionService
}

// SkillCounterUpdater updates skill-level counters after social actions.
type SkillCounterUpdater interface {
	IncrementStarCount(ctx context.Context, skillID int64) error
	DecrementStarCount(ctx context.Context, skillID int64) error
	IncrementSubscriptionCount(ctx context.Context, skillID int64) error
	DecrementSubscriptionCount(ctx context.Context, skillID int64) error
}

// SkillExistenceChecker checks whether a skill exists.
type SkillExistenceChecker interface {
	Exists(ctx context.Context, skillID int64) (bool, error)
}

// SkillStarService manages starring and unstarring skills.
// Mirrors source com.iflytek.skillhub.domain.social.SkillStarService.
type SkillStarService struct {
	starRepo       SkillStarRepository
	counterUpdater SkillCounterUpdater
	skillChecker   SkillExistenceChecker
	eventBus       eventbus.Bus
}

// NewSkillStarService creates a SkillStarService.
func NewSkillStarService(
	starRepo SkillStarRepository,
	counterUpdater SkillCounterUpdater,
	skillChecker SkillExistenceChecker,
	eventBus eventbus.Bus,
) *SkillStarService {
	return &SkillStarService{
		starRepo:       starRepo,
		counterUpdater: counterUpdater,
		skillChecker:   skillChecker,
		eventBus:       eventBus,
	}
}

// Star adds a star for a skill. Idempotent — calling twice has no effect.
func (svc *SkillStarService) Star(ctx context.Context, skillID int64, userID string) error {
	if err := svc.ensureSkillExists(ctx, skillID); err != nil {
		return err
	}

	existing, _ := svc.starRepo.FindBySkillAndUser(ctx, skillID, userID)
	if existing != nil {
		return nil // idempotent
	}

	_, err := svc.starRepo.Save(ctx, SkillStar{SkillID: skillID, UserID: userID})
	if err != nil {
		return fmt.Errorf("social: star: %w", err)
	}

	if svc.counterUpdater != nil {
		_ = svc.counterUpdater.IncrementStarCount(ctx, skillID)
	}
	svc.publishEvent(ctx, SkillStarredEvent{SkillID: skillID, UserID: userID})
	return nil
}

// Unstar removes a star. Idempotent — calling twice has no effect.
func (svc *SkillStarService) Unstar(ctx context.Context, skillID int64, userID string) error {
	if err := svc.ensureSkillExists(ctx, skillID); err != nil {
		return err
	}

	existing, err := svc.starRepo.FindBySkillAndUser(ctx, skillID, userID)
	if err != nil {
		return fmt.Errorf("social: find star: %w", err)
	}
	if existing == nil {
		return nil // idempotent
	}

	if err := svc.starRepo.Delete(ctx, existing.ID); err != nil {
		return fmt.Errorf("social: unstar: %w", err)
	}

	if svc.counterUpdater != nil {
		_ = svc.counterUpdater.DecrementStarCount(ctx, skillID)
	}
	svc.publishEvent(ctx, SkillUnstarredEvent{SkillID: skillID, UserID: userID})
	return nil
}

// IsStarred returns whether a user has starred a skill.
func (svc *SkillStarService) IsStarred(ctx context.Context, skillID int64, userID string) (bool, error) {
	existing, err := svc.starRepo.FindBySkillAndUser(ctx, skillID, userID)
	if err != nil {
		return false, err
	}
	return existing != nil, nil
}

func (svc *SkillStarService) ensureSkillExists(ctx context.Context, skillID int64) error {
	if svc.skillChecker == nil {
		return nil
	}
	exists, err := svc.skillChecker.Exists(ctx, skillID)
	if err != nil {
		return fmt.Errorf("social: check skill: %w", err)
	}
	if !exists {
		return fmt.Errorf("skill.not_found %d", skillID)
	}
	return nil
}

func (svc *SkillStarService) publishEvent(ctx context.Context, event eventbus.Event) {
	if svc.eventBus != nil {
		_ = svc.eventBus.Publish(ctx, event)
	}
}

// ---------------------------------------------------------------------------
// SkillRatingService
// ---------------------------------------------------------------------------

// SkillRatingService manages user ratings (1-5) on skills.
// Mirrors source com.iflytek.skillhub.domain.social.SkillRatingService.
type SkillRatingService struct {
	ratingRepo    SkillRatingRepository
	skillChecker   SkillExistenceChecker
	eventBus       eventbus.Bus
}

// NewSkillRatingService creates a SkillRatingService.
func NewSkillRatingService(
	ratingRepo SkillRatingRepository,
	skillChecker SkillExistenceChecker,
	eventBus eventbus.Bus,
) *SkillRatingService {
	return &SkillRatingService{
		ratingRepo:  ratingRepo,
		skillChecker: skillChecker,
		eventBus:     eventBus,
	}
}

// Rate sets or updates a user's rating for a skill. Score must be 1-5.
func (svc *SkillRatingService) Rate(ctx context.Context, skillID int64, userID string, score int16) error {
	if score < 1 || score > 5 {
		return fmt.Errorf("error.rating.score.invalid")
	}
	if err := svc.ensureSkillExists(ctx, skillID); err != nil {
		return err
	}

	existing, _ := svc.ratingRepo.FindBySkillAndUser(ctx, skillID, userID)
	if existing != nil {
		existing.Score = score
		_, err := svc.ratingRepo.Save(ctx, *existing)
		if err != nil {
			return fmt.Errorf("social: update rating: %w", err)
		}
	} else {
		_, err := svc.ratingRepo.Save(ctx, SkillRating{
			SkillID: skillID,
			UserID:  userID,
			Score:   score,
		})
		if err != nil {
			return fmt.Errorf("social: create rating: %w", err)
		}
	}

	svc.publishEvent(ctx, SkillRatedEvent{SkillID: skillID, UserID: userID, Score: score})
	return nil
}

// GetUserRating returns a user's rating for a skill, or nil if not rated.
func (svc *SkillRatingService) GetUserRating(ctx context.Context, skillID int64, userID string) (*int16, error) {
	if err := svc.ensureSkillExists(ctx, skillID); err != nil {
		return nil, err
	}
	existing, err := svc.ratingRepo.FindBySkillAndUser(ctx, skillID, userID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}
	return &existing.Score, nil
}

func (svc *SkillRatingService) ensureSkillExists(ctx context.Context, skillID int64) error {
	if svc.skillChecker == nil {
		return nil
	}
	exists, err := svc.skillChecker.Exists(ctx, skillID)
	if err != nil {
		return fmt.Errorf("social: check skill: %w", err)
	}
	if !exists {
		return fmt.Errorf("skill.not_found %d", skillID)
	}
	return nil
}

func (svc *SkillRatingService) publishEvent(ctx context.Context, event eventbus.Event) {
	if svc.eventBus != nil {
		_ = svc.eventBus.Publish(ctx, event)
	}
}

// ---------------------------------------------------------------------------
// SkillSubscriptionService
// ---------------------------------------------------------------------------

// SkillSubscriptionService manages subscribing and unsubscribing from skills.
// Mirrors source com.iflytek.skillhub.domain.social.SkillSubscriptionService.
type SkillSubscriptionService struct {
	subRepo        SkillSubscriptionRepository
	counterUpdater SkillCounterUpdater
	skillChecker   SkillExistenceChecker
	eventBus       eventbus.Bus
}

// NewSkillSubscriptionService creates a SkillSubscriptionService.
func NewSkillSubscriptionService(
	subRepo SkillSubscriptionRepository,
	counterUpdater SkillCounterUpdater,
	skillChecker SkillExistenceChecker,
	eventBus eventbus.Bus,
) *SkillSubscriptionService {
	return &SkillSubscriptionService{
		subRepo:        subRepo,
		counterUpdater: counterUpdater,
		skillChecker:   skillChecker,
		eventBus:       eventBus,
	}
}

// Subscribe subscribes a user to a skill. Idempotent.
func (svc *SkillSubscriptionService) Subscribe(ctx context.Context, skillID int64, userID string) error {
	if err := svc.ensureSkillExists(ctx, skillID); err != nil {
		return err
	}

	existing, _ := svc.subRepo.FindBySkillAndUser(ctx, skillID, userID)
	if existing != nil {
		return nil // idempotent
	}

	_, err := svc.subRepo.Save(ctx, SkillSubscription{SkillID: skillID, UserID: userID})
	if err != nil {
		return fmt.Errorf("social: subscribe: %w", err)
	}

	if svc.counterUpdater != nil {
		_ = svc.counterUpdater.IncrementSubscriptionCount(ctx, skillID)
	}
	svc.publishEvent(ctx, SkillSubscribedEvent{SkillID: skillID, UserID: userID})
	return nil
}

// Unsubscribe removes a user's subscription. Idempotent.
func (svc *SkillSubscriptionService) Unsubscribe(ctx context.Context, skillID int64, userID string) error {
	if err := svc.ensureSkillExists(ctx, skillID); err != nil {
		return err
	}

	existing, err := svc.subRepo.FindBySkillAndUser(ctx, skillID, userID)
	if err != nil {
		return fmt.Errorf("social: find subscription: %w", err)
	}
	if existing == nil {
		return nil // idempotent
	}

	if err := svc.subRepo.Delete(ctx, existing.ID); err != nil {
		return fmt.Errorf("social: unsubscribe: %w", err)
	}

	if svc.counterUpdater != nil {
		_ = svc.counterUpdater.DecrementSubscriptionCount(ctx, skillID)
	}
	svc.publishEvent(ctx, SkillUnsubscribedEvent{SkillID: skillID, UserID: userID})
	return nil
}

// IsSubscribed returns whether a user is subscribed to a skill.
func (svc *SkillSubscriptionService) IsSubscribed(ctx context.Context, skillID int64, userID string) (bool, error) {
	existing, err := svc.subRepo.FindBySkillAndUser(ctx, skillID, userID)
	if err != nil {
		return false, err
	}
	return existing != nil, nil
}

func (svc *SkillSubscriptionService) ensureSkillExists(ctx context.Context, skillID int64) error {
	if svc.skillChecker == nil {
		return nil
	}
	exists, err := svc.skillChecker.Exists(ctx, skillID)
	if err != nil {
		return fmt.Errorf("social: check skill: %w", err)
	}
	if !exists {
		return fmt.Errorf("skill.not_found %d", skillID)
	}
	return nil
}

func (svc *SkillSubscriptionService) publishEvent(ctx context.Context, event eventbus.Event) {
	if svc.eventBus != nil {
		_ = svc.eventBus.Publish(ctx, event)
	}
}
