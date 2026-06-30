// Package review manages moderation tasks for public or
// namespace-visible skill versions.
//
// Source module mapping:
//
//	skillhub-domain domain/review
//	  ReviewTask entity — moderation task for version
//	  ReviewService — submission, approval, rejection, withdrawal
//	  ReviewSubmittedEvent, ReviewApprovedEvent, ReviewRejectedEvent
//	  Optimistic status update for approval
//	  Slug conflict check with other owners' published skills
//	  Approval publishes version, updates skill latest pointer, applies visibility
//	  Rejection returns version to non-published state, captures reviewer metadata
//	  Auto-withdraw of existing pending review versions before new submit
//
// Implementation starts in Phase 06.
package review

// Service is a placeholder that will hold review workflow logic starting in Phase 06.
type Service struct{}

