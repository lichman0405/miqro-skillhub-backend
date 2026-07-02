// Package community provides skill-scoped community features: issues,
// discussions, wiki pages, change proposals, community labels, and moderation.
//
// All community objects are scoped to a skill and support linkage to
// versions, releases, namespaces, and users. Authorization is enforced
// at the service layer, not in HTTP handlers.
package community

import "time"

// ── Issue ────────────────────────────────────────────────────────────────────

// Issue represents a skill-scoped bug report, feature request, or task.
type Issue struct {
	ID              int64
	SkillID         int64
	Title           string
	Body            string
	Status          string // OPEN, CLOSED
	AssigneeID      *string
	LinkedVersionID *int64
	LinkedReleaseID *int64
	AuthorID        string
	Locked          bool
	CommentCount    int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// IssueComment represents a comment on a skill issue.
type IssueComment struct {
	ID        int64
	IssueID   int64
	AuthorID  string
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ── Discussion ───────────────────────────────────────────────────────────────

// Discussion represents a skill-scoped community discussion.
type Discussion struct {
	ID               int64
	SkillID          int64
	Title            string
	Body             string
	Category         string   // GENERAL, QA, IDEAS, ANNOUNCEMENTS
	AcceptedAnswerID *int64
	AuthorID         string
	Locked           bool
	Pinned           bool
	CommentCount     int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// DiscussionComment represents a comment on a skill discussion.
type DiscussionComment struct {
	ID           int64
	DiscussionID int64
	AuthorID     string
	Body         string
	IsAnswer     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ── Wiki Page ────────────────────────────────────────────────────────────────

// WikiPage represents a maintainer-editable documentation page scoped to a
// skill. Each page has a version history via WikiPageVersion records.
type WikiPage struct {
	ID               int64
	SkillID          int64
	Title            string
	Slug             string
	CurrentVersionID *int64
	OrderIndex       int
	AuthorID         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// WikiPageVersion represents a historical revision of a wiki page.
type WikiPageVersion struct {
	ID                   int64
	PageID               int64
	Body                 string
	Version              int
	ChangeSummary        string
	LinkedSkillVersionID *int64
	AuthorID             string
	CreatedAt            time.Time
}

// ── Change Proposal ──────────────────────────────────────────────────────────

// ChangeProposal represents a skill-native change proposal. The primary review
// object is a proposed set of package/metadata/docs changes. sourceGitRef may
// reference an external Git commit for future git-backed workflows but is not
// required.
type ChangeProposal struct {
	ID                  int64
	SkillID             int64
	Title               string
	Summary             string
	ProposedChangesJSON string // jsonb
	Status              string // OPEN, ACCEPTED, REJECTED, WITHDRAWN
	AuthorID            string
	ReviewerID          *string
	SourceGitRef        *string // future Phase 12+ git reference
	ReviewComment       string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ChangeProposalComment represents a comment on a change proposal.
type ChangeProposalComment struct {
	ID         int64
	ProposalID int64
	AuthorID   string
	Body       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ── Community Labels ─────────────────────────────────────────────────────────

// IssueLabel assigns a label_definition to a skill issue.
type IssueLabel struct {
	ID        int64
	IssueID   int64
	LabelID   int64
	CreatedAt time.Time
}

// DiscussionLabel assigns a label_definition to a skill discussion.
type DiscussionLabel struct {
	ID           int64
	DiscussionID int64
	LabelID      int64
	CreatedAt    time.Time
}

// ── Moderation ───────────────────────────────────────────────────────────────

// CommunityReport reports a community object (issue, discussion, comment, wiki
// page) for moderation, reusing the existing report/admin/audit patterns.
type CommunityReport struct {
	ID            int64
	SkillID       int64
	ObjectType    string // ISSUE, DISCUSSION, COMMENT, WIKI_PAGE
	ObjectID      int64
	ReporterID    string
	Reason        string
	Details       string
	Status        string // PENDING, RESOLVED, DISMISSED
	HandledBy     *string
	HandleComment string
	CreatedAt     time.Time
	HandledAt     *time.Time
}
