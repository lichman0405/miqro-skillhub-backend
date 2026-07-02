package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"miqro-skillhub/server/sdk/skillhub/community"
)

// noRows returns true when err is pgx.ErrNoRows.
func noRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// ── Issue Repository ─────────────────────────────────────────────────────────

type IssueRepo struct{ DB *DB }

func NewIssueRepo(db *DB) *IssueRepo { return &IssueRepo{DB: db} }

func (r *IssueRepo) Create(ctx context.Context, i community.Issue) (community.Issue, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO skill_issue (skill_id, title, body, status, assignee_id, linked_version_id, linked_release_id, author_id, locked, comment_count, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`,
		i.SkillID, i.Title, i.Body, i.Status, i.AssigneeID, i.LinkedVersionID, i.LinkedReleaseID,
		i.AuthorID, i.Locked, i.CommentCount, i.CreatedAt, i.UpdatedAt,
	).Scan(&i.ID)
	return i, err
}

func (r *IssueRepo) Update(ctx context.Context, i community.Issue) (community.Issue, error) {
	_, err := r.DB.exec(ctx,
		`UPDATE skill_issue SET title=$2, body=$3, status=$4, assignee_id=$5, linked_version_id=$6,
		 linked_release_id=$7, locked=$8, comment_count=$9, updated_at=$10 WHERE id=$1`,
		i.ID, i.Title, i.Body, i.Status, i.AssigneeID, i.LinkedVersionID, i.LinkedReleaseID,
		i.Locked, i.CommentCount, i.UpdatedAt,
	)
	return i, err
}

func (r *IssueRepo) FindByID(ctx context.Context, id int64) (*community.Issue, error) {
	var i community.Issue
	err := r.DB.queryRow(ctx,
		`SELECT id, skill_id, title, body, status, assignee_id, linked_version_id, linked_release_id,
		        author_id, locked, comment_count, created_at, updated_at
		 FROM skill_issue WHERE id=$1`, id,
	).Scan(&i.ID, &i.SkillID, &i.Title, &i.Body, &i.Status, &i.AssigneeID, &i.LinkedVersionID,
		&i.LinkedReleaseID, &i.AuthorID, &i.Locked, &i.CommentCount, &i.CreatedAt, &i.UpdatedAt)
	if noRows(err) {
		return nil, nil
	}
	return &i, err
}

func (r *IssueRepo) FindBySkillID(ctx context.Context, skillID int64, status string, offset, limit int) ([]community.Issue, error) {
	var rows pgx.Rows
	var err error
	if status == "" {
		rows, err = r.DB.query(ctx,
			`SELECT id, skill_id, title, body, status, assignee_id, linked_version_id, linked_release_id,
			        author_id, locked, comment_count, created_at, updated_at
			 FROM skill_issue WHERE skill_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			skillID, limit, offset)
	} else {
		rows, err = r.DB.query(ctx,
			`SELECT id, skill_id, title, body, status, assignee_id, linked_version_id, linked_release_id,
			        author_id, locked, comment_count, created_at, updated_at
			 FROM skill_issue WHERE skill_id=$1 AND status=$2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
			skillID, status, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIssues(rows)
}

func (r *IssueRepo) CountBySkillID(ctx context.Context, skillID int64, status string) (int64, error) {
	var count int64
	var err error
	if status == "" {
		err = r.DB.queryRow(ctx, `SELECT COUNT(*) FROM skill_issue WHERE skill_id=$1`, skillID).Scan(&count)
	} else {
		err = r.DB.queryRow(ctx, `SELECT COUNT(*) FROM skill_issue WHERE skill_id=$1 AND status=$2`, skillID, status).Scan(&count)
	}
	return count, err
}

func (r *IssueRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.exec(ctx, `DELETE FROM skill_issue WHERE id=$1`, id)
	return err
}

type IssueCommentRepo struct{ DB *DB }

func NewIssueCommentRepo(db *DB) *IssueCommentRepo { return &IssueCommentRepo{DB: db} }

func (r *IssueCommentRepo) Create(ctx context.Context, c community.IssueComment) (community.IssueComment, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO skill_issue_comment (issue_id, author_id, body, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		c.IssueID, c.AuthorID, c.Body, c.CreatedAt, c.UpdatedAt,
	).Scan(&c.ID)
	return c, err
}

func (r *IssueCommentRepo) Update(ctx context.Context, c community.IssueComment) (community.IssueComment, error) {
	_, err := r.DB.exec(ctx,
		`UPDATE skill_issue_comment SET body=$2, updated_at=$3 WHERE id=$1`,
		c.ID, c.Body, c.UpdatedAt)
	return c, err
}

func (r *IssueCommentRepo) FindByIssueID(ctx context.Context, issueID int64) ([]community.IssueComment, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, issue_id, author_id, body, created_at, updated_at
		 FROM skill_issue_comment WHERE issue_id=$1 ORDER BY created_at`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []community.IssueComment
	for rows.Next() {
		var c community.IssueComment
		if err := rows.Scan(&c.ID, &c.IssueID, &c.AuthorID, &c.Body, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *IssueCommentRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.exec(ctx, `DELETE FROM skill_issue_comment WHERE id=$1`, id)
	return err
}

// ── Discussion Repository ────────────────────────────────────────────────────

type DiscussionRepo struct{ DB *DB }

func NewDiscussionRepo(db *DB) *DiscussionRepo { return &DiscussionRepo{DB: db} }

func (r *DiscussionRepo) Create(ctx context.Context, d community.Discussion) (community.Discussion, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO skill_discussion (skill_id, title, body, category, accepted_answer_id, author_id, locked, pinned, comment_count, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		d.SkillID, d.Title, d.Body, d.Category, d.AcceptedAnswerID, d.AuthorID,
		d.Locked, d.Pinned, d.CommentCount, d.CreatedAt, d.UpdatedAt,
	).Scan(&d.ID)
	return d, err
}

func (r *DiscussionRepo) Update(ctx context.Context, d community.Discussion) (community.Discussion, error) {
	_, err := r.DB.exec(ctx,
		`UPDATE skill_discussion SET title=$2, body=$3, category=$4, accepted_answer_id=$5,
		 locked=$6, pinned=$7, comment_count=$8, updated_at=$9 WHERE id=$1`,
		d.ID, d.Title, d.Body, d.Category, d.AcceptedAnswerID,
		d.Locked, d.Pinned, d.CommentCount, d.UpdatedAt,
	)
	return d, err
}

func (r *DiscussionRepo) FindByID(ctx context.Context, id int64) (*community.Discussion, error) {
	var d community.Discussion
	err := r.DB.queryRow(ctx,
		`SELECT id, skill_id, title, body, category, accepted_answer_id, author_id,
		        locked, pinned, comment_count, created_at, updated_at
		 FROM skill_discussion WHERE id=$1`, id,
	).Scan(&d.ID, &d.SkillID, &d.Title, &d.Body, &d.Category, &d.AcceptedAnswerID,
		&d.AuthorID, &d.Locked, &d.Pinned, &d.CommentCount, &d.CreatedAt, &d.UpdatedAt)
	if noRows(err) {
		return nil, nil
	}
	return &d, err
}

func (r *DiscussionRepo) FindBySkillID(ctx context.Context, skillID int64, category string, offset, limit int) ([]community.Discussion, error) {
	var rows pgx.Rows
	var err error
	if category == "" {
		rows, err = r.DB.query(ctx,
			`SELECT id, skill_id, title, body, category, accepted_answer_id, author_id,
			        locked, pinned, comment_count, created_at, updated_at
			 FROM skill_discussion WHERE skill_id=$1 ORDER BY pinned DESC, created_at DESC LIMIT $2 OFFSET $3`,
			skillID, limit, offset)
	} else {
		rows, err = r.DB.query(ctx,
			`SELECT id, skill_id, title, body, category, accepted_answer_id, author_id,
			        locked, pinned, comment_count, created_at, updated_at
			 FROM skill_discussion WHERE skill_id=$1 AND category=$2 ORDER BY pinned DESC, created_at DESC LIMIT $3 OFFSET $4`,
			skillID, category, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []community.Discussion
	for rows.Next() {
		var d community.Discussion
		if err := rows.Scan(&d.ID, &d.SkillID, &d.Title, &d.Body, &d.Category, &d.AcceptedAnswerID,
			&d.AuthorID, &d.Locked, &d.Pinned, &d.CommentCount, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (r *DiscussionRepo) CountBySkillID(ctx context.Context, skillID int64, category string) (int64, error) {
	var count int64
	var err error
	if category == "" {
		err = r.DB.queryRow(ctx, `SELECT COUNT(*) FROM skill_discussion WHERE skill_id=$1`, skillID).Scan(&count)
	} else {
		err = r.DB.queryRow(ctx, `SELECT COUNT(*) FROM skill_discussion WHERE skill_id=$1 AND category=$2`, skillID, category).Scan(&count)
	}
	return count, err
}

func (r *DiscussionRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.exec(ctx, `DELETE FROM skill_discussion WHERE id=$1`, id)
	return err
}

type DiscCommentRepo struct{ DB *DB }

func NewDiscCommentRepo(db *DB) *DiscCommentRepo { return &DiscCommentRepo{DB: db} }

func (r *DiscCommentRepo) Create(ctx context.Context, c community.DiscussionComment) (community.DiscussionComment, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO skill_discussion_comment (discussion_id, author_id, body, is_answer, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		c.DiscussionID, c.AuthorID, c.Body, c.IsAnswer, c.CreatedAt, c.UpdatedAt,
	).Scan(&c.ID)
	return c, err
}

func (r *DiscCommentRepo) Update(ctx context.Context, c community.DiscussionComment) (community.DiscussionComment, error) {
	_, err := r.DB.exec(ctx,
		`UPDATE skill_discussion_comment SET body=$2, is_answer=$3, updated_at=$4 WHERE id=$1`,
		c.ID, c.Body, c.IsAnswer, c.UpdatedAt)
	return c, err
}

func (r *DiscCommentRepo) FindByDiscussionID(ctx context.Context, discussionID int64) ([]community.DiscussionComment, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, discussion_id, author_id, body, is_answer, created_at, updated_at
		 FROM skill_discussion_comment WHERE discussion_id=$1 ORDER BY created_at`, discussionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []community.DiscussionComment
	for rows.Next() {
		var c community.DiscussionComment
		if err := rows.Scan(&c.ID, &c.DiscussionID, &c.AuthorID, &c.Body, &c.IsAnswer, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *DiscCommentRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.exec(ctx, `DELETE FROM skill_discussion_comment WHERE id=$1`, id)
	return err
}

// ── Wiki Page Repository ─────────────────────────────────────────────────────

type WikiPageRepo struct{ DB *DB }

func NewWikiPageRepo(db *DB) *WikiPageRepo { return &WikiPageRepo{DB: db} }

func (r *WikiPageRepo) Create(ctx context.Context, p community.WikiPage) (community.WikiPage, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO skill_wiki_page (skill_id, title, slug, current_version_id, order_index, author_id, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
		p.SkillID, p.Title, p.Slug, p.CurrentVersionID, p.OrderIndex, p.AuthorID, p.CreatedAt, p.UpdatedAt,
	).Scan(&p.ID)
	return p, err
}

func (r *WikiPageRepo) Update(ctx context.Context, p community.WikiPage) (community.WikiPage, error) {
	_, err := r.DB.exec(ctx,
		`UPDATE skill_wiki_page SET title=$2, slug=$3, current_version_id=$4, order_index=$5, updated_at=$6 WHERE id=$1`,
		p.ID, p.Title, p.Slug, p.CurrentVersionID, p.OrderIndex, p.UpdatedAt)
	return p, err
}

func (r *WikiPageRepo) FindByID(ctx context.Context, id int64) (*community.WikiPage, error) {
	var p community.WikiPage
	err := r.DB.queryRow(ctx,
		`SELECT id, skill_id, title, slug, current_version_id, order_index, author_id, created_at, updated_at
		 FROM skill_wiki_page WHERE id=$1`, id,
	).Scan(&p.ID, &p.SkillID, &p.Title, &p.Slug, &p.CurrentVersionID, &p.OrderIndex,
		&p.AuthorID, &p.CreatedAt, &p.UpdatedAt)
	if noRows(err) {
		return nil, nil
	}
	return &p, err
}

func (r *WikiPageRepo) FindBySkillIDAndSlug(ctx context.Context, skillID int64, slug string) (*community.WikiPage, error) {
	var p community.WikiPage
	err := r.DB.queryRow(ctx,
		`SELECT id, skill_id, title, slug, current_version_id, order_index, author_id, created_at, updated_at
		 FROM skill_wiki_page WHERE skill_id=$1 AND slug=$2`, skillID, slug,
	).Scan(&p.ID, &p.SkillID, &p.Title, &p.Slug, &p.CurrentVersionID, &p.OrderIndex,
		&p.AuthorID, &p.CreatedAt, &p.UpdatedAt)
	if noRows(err) {
		return nil, nil
	}
	return &p, err
}

func (r *WikiPageRepo) ListBySkillID(ctx context.Context, skillID int64) ([]community.WikiPage, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, skill_id, title, slug, current_version_id, order_index, author_id, created_at, updated_at
		 FROM skill_wiki_page WHERE skill_id=$1 ORDER BY order_index`, skillID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []community.WikiPage
	for rows.Next() {
		var p community.WikiPage
		if err := rows.Scan(&p.ID, &p.SkillID, &p.Title, &p.Slug, &p.CurrentVersionID, &p.OrderIndex,
			&p.AuthorID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *WikiPageRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.exec(ctx, `DELETE FROM skill_wiki_page WHERE id=$1`, id)
	return err
}

type WikiVersionRepo struct{ DB *DB }

func NewWikiVersionRepo(db *DB) *WikiVersionRepo { return &WikiVersionRepo{DB: db} }

func (r *WikiVersionRepo) Create(ctx context.Context, v community.WikiPageVersion) (community.WikiPageVersion, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO skill_wiki_page_version (page_id, body, version, change_summary, linked_skill_version_id, author_id, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		v.PageID, v.Body, v.Version, v.ChangeSummary, v.LinkedSkillVersionID, v.AuthorID, v.CreatedAt,
	).Scan(&v.ID)
	return v, err
}

func (r *WikiVersionRepo) FindByPageID(ctx context.Context, pageID int64) ([]community.WikiPageVersion, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, page_id, body, version, change_summary, linked_skill_version_id, author_id, created_at
		 FROM skill_wiki_page_version WHERE page_id=$1 ORDER BY version DESC`, pageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []community.WikiPageVersion
	for rows.Next() {
		var v community.WikiPageVersion
		if err := rows.Scan(&v.ID, &v.PageID, &v.Body, &v.Version, &v.ChangeSummary,
			&v.LinkedSkillVersionID, &v.AuthorID, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// ── Change Proposal Repository ───────────────────────────────────────────────

type ChangeProposalRepo struct{ DB *DB }

func NewChangeProposalRepo(db *DB) *ChangeProposalRepo { return &ChangeProposalRepo{DB: db} }

func (r *ChangeProposalRepo) Create(ctx context.Context, p community.ChangeProposal) (community.ChangeProposal, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO skill_change_proposal (skill_id, title, summary, proposed_changes_json, status, author_id, reviewer_id, source_git_ref, review_comment, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		p.SkillID, p.Title, p.Summary, p.ProposedChangesJSON, p.Status, p.AuthorID,
		p.ReviewerID, p.SourceGitRef, p.ReviewComment, p.CreatedAt, p.UpdatedAt,
	).Scan(&p.ID)
	return p, err
}

func (r *ChangeProposalRepo) Update(ctx context.Context, p community.ChangeProposal) (community.ChangeProposal, error) {
	_, err := r.DB.exec(ctx,
		`UPDATE skill_change_proposal SET title=$2, summary=$3, proposed_changes_json=$4, status=$5,
		 reviewer_id=$6, source_git_ref=$7, review_comment=$8, updated_at=$9 WHERE id=$1`,
		p.ID, p.Title, p.Summary, p.ProposedChangesJSON, p.Status,
		p.ReviewerID, p.SourceGitRef, p.ReviewComment, p.UpdatedAt,
	)
	return p, err
}

func (r *ChangeProposalRepo) FindByID(ctx context.Context, id int64) (*community.ChangeProposal, error) {
	var p community.ChangeProposal
	err := r.DB.queryRow(ctx,
		`SELECT id, skill_id, title, summary, proposed_changes_json, status, author_id, reviewer_id,
		        source_git_ref, review_comment, created_at, updated_at
		 FROM skill_change_proposal WHERE id=$1`, id,
	).Scan(&p.ID, &p.SkillID, &p.Title, &p.Summary, &p.ProposedChangesJSON, &p.Status,
		&p.AuthorID, &p.ReviewerID, &p.SourceGitRef, &p.ReviewComment, &p.CreatedAt, &p.UpdatedAt)
	if noRows(err) {
		return nil, nil
	}
	return &p, err
}

func (r *ChangeProposalRepo) FindBySkillID(ctx context.Context, skillID int64, status string, offset, limit int) ([]community.ChangeProposal, error) {
	var rows pgx.Rows
	var err error
	if status == "" {
		rows, err = r.DB.query(ctx,
			`SELECT id, skill_id, title, summary, proposed_changes_json, status, author_id, reviewer_id,
			        source_git_ref, review_comment, created_at, updated_at
			 FROM skill_change_proposal WHERE skill_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			skillID, limit, offset)
	} else {
		rows, err = r.DB.query(ctx,
			`SELECT id, skill_id, title, summary, proposed_changes_json, status, author_id, reviewer_id,
			        source_git_ref, review_comment, created_at, updated_at
			 FROM skill_change_proposal WHERE skill_id=$1 AND status=$2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
			skillID, status, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []community.ChangeProposal
	for rows.Next() {
		var p community.ChangeProposal
		if err := rows.Scan(&p.ID, &p.SkillID, &p.Title, &p.Summary, &p.ProposedChangesJSON, &p.Status,
			&p.AuthorID, &p.ReviewerID, &p.SourceGitRef, &p.ReviewComment, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *ChangeProposalRepo) CountBySkillID(ctx context.Context, skillID int64, status string) (int64, error) {
	var count int64
	var err error
	if status == "" {
		err = r.DB.queryRow(ctx, `SELECT COUNT(*) FROM skill_change_proposal WHERE skill_id=$1`, skillID).Scan(&count)
	} else {
		err = r.DB.queryRow(ctx, `SELECT COUNT(*) FROM skill_change_proposal WHERE skill_id=$1 AND status=$2`, skillID, status).Scan(&count)
	}
	return count, err
}

func (r *ChangeProposalRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.exec(ctx, `DELETE FROM skill_change_proposal WHERE id=$1`, id)
	return err
}

// ── Label Repositories ───────────────────────────────────────────────────────

type IssueLabelRepo struct{ DB *DB }

func NewIssueLabelRepo(db *DB) *IssueLabelRepo { return &IssueLabelRepo{DB: db} }

func (r *IssueLabelRepo) Add(ctx context.Context, l community.IssueLabel) (community.IssueLabel, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO issue_label (issue_id, label_id, created_at) VALUES ($1,$2,$3)
		 ON CONFLICT (issue_id, label_id) DO NOTHING RETURNING id`,
		l.IssueID, l.LabelID, l.CreatedAt,
	).Scan(&l.ID)
	if noRows(err) {
		// Already exists — return existing.
		return l, nil
	}
	return l, err
}

func (r *IssueLabelRepo) Remove(ctx context.Context, issueID, labelID int64) error {
	_, err := r.DB.exec(ctx, `DELETE FROM issue_label WHERE issue_id=$1 AND label_id=$2`, issueID, labelID)
	return err
}

func (r *IssueLabelRepo) FindByIssueID(ctx context.Context, issueID int64) ([]community.IssueLabel, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, issue_id, label_id, created_at FROM issue_label WHERE issue_id=$1`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []community.IssueLabel
	for rows.Next() {
		var l community.IssueLabel
		if err := rows.Scan(&l.ID, &l.IssueID, &l.LabelID, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, nil
}

type DiscussionLabelRepo struct{ DB *DB }

func NewDiscussionLabelRepo(db *DB) *DiscussionLabelRepo { return &DiscussionLabelRepo{DB: db} }

func (r *DiscussionLabelRepo) Add(ctx context.Context, l community.DiscussionLabel) (community.DiscussionLabel, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO discussion_label (discussion_id, label_id, created_at) VALUES ($1,$2,$3)
		 ON CONFLICT (discussion_id, label_id) DO NOTHING RETURNING id`,
		l.DiscussionID, l.LabelID, l.CreatedAt,
	).Scan(&l.ID)
	if noRows(err) {
		return l, nil
	}
	return l, err
}

func (r *DiscussionLabelRepo) Remove(ctx context.Context, discussionID, labelID int64) error {
	_, err := r.DB.exec(ctx, `DELETE FROM discussion_label WHERE discussion_id=$1 AND label_id=$2`, discussionID, labelID)
	return err
}

func (r *DiscussionLabelRepo) FindByDiscussionID(ctx context.Context, discussionID int64) ([]community.DiscussionLabel, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, discussion_id, label_id, created_at FROM discussion_label WHERE discussion_id=$1`, discussionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []community.DiscussionLabel
	for rows.Next() {
		var l community.DiscussionLabel
		if err := rows.Scan(&l.ID, &l.DiscussionID, &l.LabelID, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, nil
}

// ── Community Report Repository ──────────────────────────────────────────────

type CommunityReportRepo struct{ DB *DB }

func NewCommunityReportRepo(db *DB) *CommunityReportRepo { return &CommunityReportRepo{DB: db} }

func (r *CommunityReportRepo) Create(ctx context.Context, rep community.CommunityReport) (community.CommunityReport, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO community_report (skill_id, object_type, object_id, reporter_id, reason, details, status, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
		rep.SkillID, rep.ObjectType, rep.ObjectID, rep.ReporterID,
		rep.Reason, rep.Details, rep.Status, rep.CreatedAt,
	).Scan(&rep.ID)
	return rep, err
}

func (r *CommunityReportRepo) Update(ctx context.Context, rep community.CommunityReport) (community.CommunityReport, error) {
	_, err := r.DB.exec(ctx,
		`UPDATE community_report SET status=$2, handled_by=$3, handle_comment=$4, handled_at=$5 WHERE id=$1`,
		rep.ID, rep.Status, rep.HandledBy, rep.HandleComment, rep.HandledAt)
	return rep, err
}

func (r *CommunityReportRepo) FindByID(ctx context.Context, id int64) (*community.CommunityReport, error) {
	var rep community.CommunityReport
	err := r.DB.queryRow(ctx,
		`SELECT id, skill_id, object_type, object_id, reporter_id, reason, details, status,
		        handled_by, handle_comment, created_at, handled_at
		 FROM community_report WHERE id=$1`, id,
	).Scan(&rep.ID, &rep.SkillID, &rep.ObjectType, &rep.ObjectID, &rep.ReporterID,
		&rep.Reason, &rep.Details, &rep.Status, &rep.HandledBy, &rep.HandleComment,
		&rep.CreatedAt, &rep.HandledAt)
	if noRows(err) {
		return nil, nil
	}
	return &rep, err
}

func (r *CommunityReportRepo) FindByStatus(ctx context.Context, status string, offset, limit int) ([]community.CommunityReport, error) {
	var rows pgx.Rows
	var err error
	if status == "" {
		rows, err = r.DB.query(ctx,
			`SELECT id, skill_id, object_type, object_id, reporter_id, reason, details, status,
			        handled_by, handle_comment, created_at, handled_at
			 FROM community_report ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	} else {
		rows, err = r.DB.query(ctx,
			`SELECT id, skill_id, object_type, object_id, reporter_id, reason, details, status,
			        handled_by, handle_comment, created_at, handled_at
			 FROM community_report WHERE status=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			status, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCommunityReports(rows)
}

func (r *CommunityReportRepo) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	var err error
	if status == "" {
		err = r.DB.queryRow(ctx, `SELECT COUNT(*) FROM community_report`).Scan(&count)
	} else {
		err = r.DB.queryRow(ctx, `SELECT COUNT(*) FROM community_report WHERE status=$1`, status).Scan(&count)
	}
	return count, err
}

func (r *CommunityReportRepo) FindByObject(ctx context.Context, objectType string, objectID int64) ([]community.CommunityReport, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, skill_id, object_type, object_id, reporter_id, reason, details, status,
		        handled_by, handle_comment, created_at, handled_at
		 FROM community_report WHERE object_type=$1 AND object_id=$2 ORDER BY created_at DESC`,
		objectType, objectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCommunityReports(rows)
}

// ── Change Proposal Comment Repository ───────────────────────────────────────

type ProposalCommentRepo struct{ DB *DB }

func NewProposalCommentRepo(db *DB) *ProposalCommentRepo { return &ProposalCommentRepo{DB: db} }

func (r *ProposalCommentRepo) Create(ctx context.Context, c community.ChangeProposalComment) (community.ChangeProposalComment, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO skill_change_proposal_comment (proposal_id, author_id, body, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		c.ProposalID, c.AuthorID, c.Body, c.CreatedAt, c.UpdatedAt,
	).Scan(&c.ID)
	return c, err
}

func (r *ProposalCommentRepo) FindByProposalID(ctx context.Context, proposalID int64) ([]community.ChangeProposalComment, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, proposal_id, author_id, body, created_at, updated_at
		 FROM skill_change_proposal_comment WHERE proposal_id=$1 ORDER BY created_at`, proposalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []community.ChangeProposalComment
	for rows.Next() {
		var c community.ChangeProposalComment
		if err := rows.Scan(&c.ID, &c.ProposalID, &c.AuthorID, &c.Body, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// ── Version/Release lookups for cross-skill validation ─────────────────────

type CommunityVersionLookup struct{ DB *DB }

func NewCommunityVersionLookup(db *DB) *CommunityVersionLookup {
	return &CommunityVersionLookup{DB: db}
}

func (l *CommunityVersionLookup) FindByID(ctx context.Context, id int64) (*community.VersionRef, error) {
	var v community.VersionRef
	err := l.DB.queryRow(ctx,
		`SELECT id, skill_id FROM skill_version WHERE id=$1`, id,
	).Scan(&v.ID, &v.SkillID)
	if noRows(err) {
		return nil, nil
	}
	return &v, err
}

type CommunityReleaseLookup struct{ DB *DB }

func NewCommunityReleaseLookup(db *DB) *CommunityReleaseLookup {
	return &CommunityReleaseLookup{DB: db}
}

func (l *CommunityReleaseLookup) FindByID(ctx context.Context, id int64) (*community.ReleaseRef, error) {
	var r community.ReleaseRef
	err := l.DB.queryRow(ctx,
		`SELECT id, skill_id FROM skill_release WHERE id=$1`, id,
	).Scan(&r.ID, &r.SkillID)
	if noRows(err) {
		return nil, nil
	}
	return &r, err
}

// ── Community Search Repository ─────────────────────────────────────────────

type CommunitySearchRepo struct{ DB *DB }

func NewCommunitySearchRepo(db *DB) *CommunitySearchRepo {
	return &CommunitySearchRepo{DB: db}
}

func (r *CommunitySearchRepo) Search(ctx context.Context, skillID int64, query string, types []string, offset, limit int) ([]community.SearchResultItem, error) {
	sqlParts, args := buildCommunitySearchParts(query, types)
	if len(sqlParts) == 0 {
		return nil, nil
	}

	// $1 = skillID; args has one entry per included table, starting at $2.
	// LIMIT = next after all query args, OFFSET = after LIMIT.
	nextParam := len(args) + 2
	sql := strings.Join(sqlParts, " UNION ALL ") + fmt.Sprintf(" ORDER BY type, title LIMIT $%d OFFSET $%d", nextParam, nextParam+1)
	args = append(args, limit, offset)

	allArgs := append([]any{skillID}, args...)
	rows, err := r.DB.query(ctx, sql, allArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []community.SearchResultItem
	for rows.Next() {
		var item community.SearchResultItem
		if err := rows.Scan(&item.Type, &item.ID, &item.SkillID, &item.Title, &item.Snippet); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

// buildCommunitySearchParts builds UNION ALL subqueries for community search.
// $1 is always skill_id. Query parameters start at $2 and increment per table.
// Each included table contributes one arg (the query string).
func buildCommunitySearchParts(query string, types []string) (sqlParts []string, args []any) {
	useTable := func(t string) bool {
		if len(types) == 0 {
			return true
		}
		for _, tt := range types {
			if tt == t {
				return true
			}
		}
		return false
	}

	next := 2 // $1 = skill_id, query params start at $2

	if useTable("ISSUE") {
		sqlParts = append(sqlParts, fmt.Sprintf(
			`SELECT 'ISSUE' AS type, id, skill_id, title, COALESCE(body,'') AS snippet FROM skill_issue WHERE skill_id=$1 AND ($%d::text='' OR title ILIKE '%%'||$%d::text||'%%' OR body ILIKE '%%'||$%d::text||'%%')`,
			next, next, next))
		args = append(args, query)
		next++
	}
	if useTable("DISCUSSION") {
		sqlParts = append(sqlParts, fmt.Sprintf(
			`SELECT 'DISCUSSION' AS type, id, skill_id, title, COALESCE(body,'') AS snippet FROM skill_discussion WHERE skill_id=$1 AND ($%d::text='' OR title ILIKE '%%'||$%d::text||'%%' OR body ILIKE '%%'||$%d::text||'%%')`,
			next, next, next))
		args = append(args, query)
		next++
	}
	if useTable("WIKI_PAGE") {
		sqlParts = append(sqlParts, fmt.Sprintf(
			`SELECT 'WIKI_PAGE' AS type, sp.id, sp.skill_id, sp.title, COALESCE(wv.body,'') AS snippet FROM skill_wiki_page sp LEFT JOIN skill_wiki_page_version wv ON wv.id = sp.current_version_id WHERE sp.skill_id=$1 AND ($%d::text='' OR sp.title ILIKE '%%'||$%d::text||'%%' OR wv.body ILIKE '%%'||$%d::text||'%%')`,
			next, next, next))
		args = append(args, query)
		next++
	}
	if useTable("PROPOSAL") {
		sqlParts = append(sqlParts, fmt.Sprintf(
			`SELECT 'PROPOSAL' AS type, id, skill_id, title, COALESCE(summary,'') AS snippet FROM skill_change_proposal WHERE skill_id=$1 AND ($%d::text='' OR title ILIKE '%%'||$%d::text||'%%' OR summary ILIKE '%%'||$%d::text||'%%')`,
			next, next, next))
		args = append(args, query)
		next++
	}

	return sqlParts, args
}

func (r *CommunitySearchRepo) Count(ctx context.Context, skillID int64, query string, types []string) (int64, error) {
	var total int64
	for _, q := range buildCommunitySearchCountQueries(skillID, query, types) {
		var c int64
		if err := r.DB.queryRow(ctx, q.sql, q.args...).Scan(&c); err != nil {
			return 0, fmt.Errorf("community search count (%s): %w", q.table, err)
		}
		total += c
	}
	return total, nil
}

// countQuery holds a single count query for one table.
type countQuery struct {
	table string
	sql   string
	args  []any
}

// buildCommunitySearchCountQueries builds one COUNT(*) query per matching table.
// It applies the same skill_id, query (ILIKE), and types filtering as Search.
func buildCommunitySearchCountQueries(skillID int64, query string, types []string) []countQuery {
	useTable := func(t string) bool {
		if len(types) == 0 {
			return true
		}
		for _, tt := range types {
			if tt == t {
				return true
			}
		}
		return false
	}

	var queries []countQuery

	if useTable("ISSUE") {
		q := countQuery{table: "ISSUE"}
		if query == "" {
			q.sql = `SELECT COUNT(*) FROM skill_issue WHERE skill_id=$1`
			q.args = []any{skillID}
		} else {
			q.sql = `SELECT COUNT(*) FROM skill_issue WHERE skill_id=$1 AND (title ILIKE '%' || $2 || '%' OR body ILIKE '%' || $2 || '%')`
			q.args = []any{skillID, query}
		}
		queries = append(queries, q)
	}

	if useTable("DISCUSSION") {
		q := countQuery{table: "DISCUSSION"}
		if query == "" {
			q.sql = `SELECT COUNT(*) FROM skill_discussion WHERE skill_id=$1`
			q.args = []any{skillID}
		} else {
			q.sql = `SELECT COUNT(*) FROM skill_discussion WHERE skill_id=$1 AND (title ILIKE '%' || $2 || '%' OR body ILIKE '%' || $2 || '%')`
			q.args = []any{skillID, query}
		}
		queries = append(queries, q)
	}

	if useTable("WIKI_PAGE") {
		q := countQuery{table: "WIKI_PAGE"}
		if query == "" {
			q.sql = `SELECT COUNT(*) FROM skill_wiki_page WHERE skill_id=$1`
			q.args = []any{skillID}
		} else {
			q.sql = `SELECT COUNT(*) FROM skill_wiki_page sp LEFT JOIN skill_wiki_page_version wv ON wv.id = sp.current_version_id WHERE sp.skill_id=$1 AND (sp.title ILIKE '%' || $2 || '%' OR wv.body ILIKE '%' || $2 || '%')`
			q.args = []any{skillID, query}
		}
		queries = append(queries, q)
	}

	if useTable("PROPOSAL") {
		q := countQuery{table: "PROPOSAL"}
		if query == "" {
			q.sql = `SELECT COUNT(*) FROM skill_change_proposal WHERE skill_id=$1`
			q.args = []any{skillID}
		} else {
			q.sql = `SELECT COUNT(*) FROM skill_change_proposal WHERE skill_id=$1 AND (title ILIKE '%' || $2 || '%' OR summary ILIKE '%' || $2 || '%')`
			q.args = []any{skillID, query}
		}
		queries = append(queries, q)
	}

	return queries
}

// ── Scan helpers ─────────────────────────────────────────────────────────────

func scanIssues(rows pgx.Rows) ([]community.Issue, error) {
	var out []community.Issue
	for rows.Next() {
		var i community.Issue
		if err := rows.Scan(&i.ID, &i.SkillID, &i.Title, &i.Body, &i.Status, &i.AssigneeID,
			&i.LinkedVersionID, &i.LinkedReleaseID, &i.AuthorID, &i.Locked, &i.CommentCount,
			&i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, nil
}

func scanCommunityReports(rows pgx.Rows) ([]community.CommunityReport, error) {
	var out []community.CommunityReport
	for rows.Next() {
		var rep community.CommunityReport
		if err := rows.Scan(&rep.ID, &rep.SkillID, &rep.ObjectType, &rep.ObjectID, &rep.ReporterID,
			&rep.Reason, &rep.Details, &rep.Status, &rep.HandledBy, &rep.HandleComment,
			&rep.CreatedAt, &rep.HandledAt); err != nil {
			return nil, err
		}
		out = append(out, rep)
	}
	return out, nil
}
