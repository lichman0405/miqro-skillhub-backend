package review

// ReviewSubmittedEvent is emitted when a review task is created.
type ReviewSubmittedEvent struct {
	TaskID         int64
	SkillID        int64
	SkillVersionID int64
	SubmittedBy    string
	NamespaceID    int64
}

func (e ReviewSubmittedEvent) EventName() string { return "review.submitted" }

// ReviewApprovedEvent is emitted when a review task is approved.
type ReviewApprovedEvent struct {
	TaskID         int64
	SkillID        int64
	SkillVersionID int64
	ReviewedBy     string
	SubmittedBy    string
}

func (e ReviewApprovedEvent) EventName() string { return "review.approved" }

// ReviewRejectedEvent is emitted when a review task is rejected.
type ReviewRejectedEvent struct {
	TaskID         int64
	SkillID        int64
	SkillVersionID int64
	ReviewedBy     string
	SubmittedBy    string
	Comment        string
}

func (e ReviewRejectedEvent) EventName() string { return "review.rejected" }
