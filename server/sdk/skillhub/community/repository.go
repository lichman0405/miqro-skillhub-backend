package community

import "context"

// ── Issue Repository ─────────────────────────────────────────────────────────

// IssueRepository defines the persistence contract for skill issues.
type IssueRepository interface {
	Create(ctx context.Context, i Issue) (Issue, error)
	Update(ctx context.Context, i Issue) (Issue, error)
	FindByID(ctx context.Context, id int64) (*Issue, error)
	FindBySkillID(ctx context.Context, skillID int64, status string, offset, limit int) ([]Issue, error)
	CountBySkillID(ctx context.Context, skillID int64, status string) (int64, error)
	Delete(ctx context.Context, id int64) error
}

// IssueCommentRepository defines the persistence contract for issue comments.
type IssueCommentRepository interface {
	Create(ctx context.Context, c IssueComment) (IssueComment, error)
	Update(ctx context.Context, c IssueComment) (IssueComment, error)
	FindByIssueID(ctx context.Context, issueID int64) ([]IssueComment, error)
	Delete(ctx context.Context, id int64) error
}

// ── Discussion Repository ────────────────────────────────────────────────────

// DiscussionRepository defines the persistence contract for skill discussions.
type DiscussionRepository interface {
	Create(ctx context.Context, d Discussion) (Discussion, error)
	Update(ctx context.Context, d Discussion) (Discussion, error)
	FindByID(ctx context.Context, id int64) (*Discussion, error)
	FindBySkillID(ctx context.Context, skillID int64, category string, offset, limit int) ([]Discussion, error)
	CountBySkillID(ctx context.Context, skillID int64, category string) (int64, error)
	Delete(ctx context.Context, id int64) error
}

// DiscussionCommentRepository defines the persistence contract.
type DiscussionCommentRepository interface {
	Create(ctx context.Context, c DiscussionComment) (DiscussionComment, error)
	Update(ctx context.Context, c DiscussionComment) (DiscussionComment, error)
	FindByDiscussionID(ctx context.Context, discussionID int64) ([]DiscussionComment, error)
	Delete(ctx context.Context, id int64) error
}

// ── Wiki Page Repository ─────────────────────────────────────────────────────

// WikiPageRepository defines the persistence contract for skill wiki pages.
type WikiPageRepository interface {
	Create(ctx context.Context, p WikiPage) (WikiPage, error)
	Update(ctx context.Context, p WikiPage) (WikiPage, error)
	FindByID(ctx context.Context, id int64) (*WikiPage, error)
	FindBySkillIDAndSlug(ctx context.Context, skillID int64, slug string) (*WikiPage, error)
	ListBySkillID(ctx context.Context, skillID int64) ([]WikiPage, error)
	Delete(ctx context.Context, id int64) error
}

// WikiPageVersionRepository defines the persistence contract for page versions.
type WikiPageVersionRepository interface {
	Create(ctx context.Context, v WikiPageVersion) (WikiPageVersion, error)
	FindByPageID(ctx context.Context, pageID int64) ([]WikiPageVersion, error)
}

// ── Change Proposal Repository ───────────────────────────────────────────────

// ChangeProposalRepository defines the persistence contract.
type ChangeProposalRepository interface {
	Create(ctx context.Context, p ChangeProposal) (ChangeProposal, error)
	Update(ctx context.Context, p ChangeProposal) (ChangeProposal, error)
	FindByID(ctx context.Context, id int64) (*ChangeProposal, error)
	FindBySkillID(ctx context.Context, skillID int64, status string, offset, limit int) ([]ChangeProposal, error)
	CountBySkillID(ctx context.Context, skillID int64, status string) (int64, error)
	Delete(ctx context.Context, id int64) error
}

// ChangeProposalCommentRepository defines the persistence contract.
type ChangeProposalCommentRepository interface {
	Create(ctx context.Context, c ChangeProposalComment) (ChangeProposalComment, error)
	FindByProposalID(ctx context.Context, proposalID int64) ([]ChangeProposalComment, error)
}

// ── Community Label Repositories ─────────────────────────────────────────────

// IssueLabelRepository manages issue-label assignments.
type IssueLabelRepository interface {
	Add(ctx context.Context, l IssueLabel) (IssueLabel, error)
	Remove(ctx context.Context, issueID, labelID int64) error
	FindByIssueID(ctx context.Context, issueID int64) ([]IssueLabel, error)
}

// DiscussionLabelRepository manages discussion-label assignments.
type DiscussionLabelRepository interface {
	Add(ctx context.Context, l DiscussionLabel) (DiscussionLabel, error)
	Remove(ctx context.Context, discussionID, labelID int64) error
	FindByDiscussionID(ctx context.Context, discussionID int64) ([]DiscussionLabel, error)
}

// ── Moderation Repository ────────────────────────────────────────────────────

// CommunityReportRepository defines the persistence contract for community
// object reports, reusing the existing report/admin/audit patterns.
type CommunityReportRepository interface {
	Create(ctx context.Context, r CommunityReport) (CommunityReport, error)
	Update(ctx context.Context, r CommunityReport) (CommunityReport, error)
	FindByID(ctx context.Context, id int64) (*CommunityReport, error)
	FindByStatus(ctx context.Context, status string, offset, limit int) ([]CommunityReport, error)
	CountByStatus(ctx context.Context, status string) (int64, error)
	FindByObject(ctx context.Context, objectType string, objectID int64) ([]CommunityReport, error)
}
