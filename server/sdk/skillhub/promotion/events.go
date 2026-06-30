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

// PromotionRejectedEvent is emitted when a promotion request is rejected.
type PromotionRejectedEvent struct {
	RequestID     int64
	SourceSkillID int64
	ReviewedBy    string
	SubmittedBy   string
	Comment       string
}

func (e PromotionRejectedEvent) EventName() string { return "promotion.rejected" }

// SkillPublishedEvent is emitted when a skill is published via promotion.
type SkillPublishedEvent struct {
	SkillID        int64
	SkillVersionID int64
	PublishedBy    string
}

func (e SkillPublishedEvent) EventName() string { return "skill.published" }
