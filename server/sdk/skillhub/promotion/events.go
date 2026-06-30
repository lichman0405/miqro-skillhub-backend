package promotion

// PromotionSubmittedEvent is emitted when a promotion request is created.
type PromotionSubmittedEvent struct {
	RequestID       int64
	SourceSkillID   int64
	SourceVersionID int64
	SubmittedBy     string
}

func (e PromotionSubmittedEvent) EventName() string { return "promotion.submitted" }

// PromotionApprovedEvent is emitted when a promotion request is approved.
type PromotionApprovedEvent struct {
	RequestID     int64
	SourceSkillID int64
	ReviewedBy    string
	SubmittedBy   string
}

func (e PromotionApprovedEvent) EventName() string { return "promotion.approved" }

// GetSubmittedBy satisfies governance.PromotionApprovedEvent.
func (e PromotionApprovedEvent) GetSubmittedBy() string { return e.SubmittedBy }

// GetRequestID satisfies governance.PromotionApprovedEvent.
func (e PromotionApprovedEvent) GetRequestID() int64 { return e.RequestID }

// PromotionRejectedEvent is emitted when a promotion request is rejected.
type PromotionRejectedEvent struct {
	RequestID     int64
	SourceSkillID int64
	ReviewedBy    string
	SubmittedBy   string
	Comment       string
}

func (e PromotionRejectedEvent) EventName() string { return "promotion.rejected" }

// GetSubmittedBy satisfies governance.PromotionRejectedEvent.
func (e PromotionRejectedEvent) GetSubmittedBy() string { return e.SubmittedBy }

// GetRequestID satisfies governance.PromotionRejectedEvent.
func (e PromotionRejectedEvent) GetRequestID() int64 { return e.RequestID }


// SkillPublishedEvent is emitted when a skill is published via promotion.
type SkillPublishedEvent struct {
	SkillID        int64
	SkillVersionID int64
	PublishedBy    string
}

func (e SkillPublishedEvent) EventName() string { return "skill.published" }
