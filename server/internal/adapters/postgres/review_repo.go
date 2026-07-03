package postgres

import (
	"context"
	"time"

	"miqro-skillhub/server/sdk/skillhub/review"
)

// ReviewTaskRepo implements review.ReviewTaskRepository.
type ReviewTaskRepo struct{ *DB }

// Compile-time assertion.
var _ review.ReviewTaskRepository = (*ReviewTaskRepo)(nil)

func NewReviewTaskRepo(db *DB) *ReviewTaskRepo { return &ReviewTaskRepo{DB: db} }

func (r *ReviewTaskRepo) Save(ctx context.Context, t review.ReviewTask) (review.ReviewTask, error) {
	if t.SubmittedAt.IsZero() {
		t.SubmittedAt = time.Now()
	}

	err := r.queryRow(ctx,
		`INSERT INTO review_task (skill_version_id, namespace_id, status, version, submitted_by, reviewed_by, review_comment, submitted_at, reviewed_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING id, skill_version_id, namespace_id, status, version, submitted_by, reviewed_by, review_comment, submitted_at, reviewed_at`,
		t.SkillVersionID, t.NamespaceID, t.Status, t.Version, t.SubmittedBy, t.ReviewedBy, t.ReviewComment, t.SubmittedAt, t.ReviewedAt,
	).Scan(&t.ID, &t.SkillVersionID, &t.NamespaceID, &t.Status, &t.Version, &t.SubmittedBy, &t.ReviewedBy, &t.ReviewComment, &t.SubmittedAt, &t.ReviewedAt)
	if err != nil {
		return review.ReviewTask{}, err
	}
	return t, nil
}

func (r *ReviewTaskRepo) FindByID(ctx context.Context, id int64) (*review.ReviewTask, error) {
	var t review.ReviewTask
	err := r.queryRow(ctx,
		`SELECT id, skill_version_id, namespace_id, status, version, submitted_by, reviewed_by, review_comment, submitted_at, reviewed_at
		 FROM review_task WHERE id = $1`, id,
	).Scan(&t.ID, &t.SkillVersionID, &t.NamespaceID, &t.Status, &t.Version, &t.SubmittedBy, &t.ReviewedBy, &t.ReviewComment, &t.SubmittedAt, &t.ReviewedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *ReviewTaskRepo) FindByVersionIDAndStatus(ctx context.Context, versionID int64, status string) (*review.ReviewTask, error) {
	var t review.ReviewTask
	err := r.queryRow(ctx,
		`SELECT id, skill_version_id, namespace_id, status, version, submitted_by, reviewed_by, review_comment, submitted_at, reviewed_at
		 FROM review_task WHERE skill_version_id = $1 AND status = $2`, versionID, status,
	).Scan(&t.ID, &t.SkillVersionID, &t.NamespaceID, &t.Status, &t.Version, &t.SubmittedBy, &t.ReviewedBy, &t.ReviewComment, &t.SubmittedAt, &t.ReviewedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *ReviewTaskRepo) FindByStatus(ctx context.Context, status string) ([]review.ReviewTask, error) {
	rows, err := r.query(ctx,
		`SELECT id, skill_version_id, namespace_id, status, version, submitted_by, reviewed_by, review_comment, submitted_at, reviewed_at
		 FROM review_task WHERE status = $1 ORDER BY submitted_at DESC`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReviewTasks(rows)
}

func (r *ReviewTaskRepo) FindByStatusPaged(ctx context.Context, status string, page int, size int) ([]review.ReviewTask, bool, error) {
	rows, err := r.query(ctx,
		`SELECT id, skill_version_id, namespace_id, status, version, submitted_by, reviewed_by, review_comment, submitted_at, reviewed_at
		 FROM review_task WHERE status = $1 ORDER BY submitted_at DESC, id DESC LIMIT $2 OFFSET $3`,
		status, size+1, page*size)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	tasks, err := scanReviewTasks(rows)
	if err != nil {
		return nil, false, err
	}
	if len(tasks) > size {
		return tasks[:size], true, nil
	}
	return tasks, false, nil
}

func (r *ReviewTaskRepo) FindByNamespaceIDsAndStatusPaged(ctx context.Context, namespaceIDs []int64, status string, page int, size int) ([]review.ReviewTask, bool, error) {
	if len(namespaceIDs) == 0 {
		return nil, false, nil
	}
	rows, err := r.query(ctx,
		`SELECT id, skill_version_id, namespace_id, status, version, submitted_by, reviewed_by, review_comment, submitted_at, reviewed_at
		 FROM review_task WHERE namespace_id = ANY($1) AND status = $2 ORDER BY submitted_at DESC, id DESC LIMIT $3 OFFSET $4`,
		namespaceIDs, status, size+1, page*size)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	tasks, err := scanReviewTasks(rows)
	if err != nil {
		return nil, false, err
	}
	if len(tasks) > size {
		return tasks[:size], true, nil
	}
	return tasks, false, nil
}

func (r *ReviewTaskRepo) FindByNamespaceIDAndStatus(ctx context.Context, namespaceID int64, status string) ([]review.ReviewTask, error) {
	rows, err := r.query(ctx,
		`SELECT id, skill_version_id, namespace_id, status, version, submitted_by, reviewed_by, review_comment, submitted_at, reviewed_at
		 FROM review_task WHERE namespace_id = $1 AND status = $2 ORDER BY submitted_at DESC`, namespaceID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReviewTasks(rows)
}

func (r *ReviewTaskRepo) FindBySubmittedByAndStatus(ctx context.Context, submittedBy string, status string) ([]review.ReviewTask, error) {
	rows, err := r.query(ctx,
		`SELECT id, skill_version_id, namespace_id, status, version, submitted_by, reviewed_by, review_comment, submitted_at, reviewed_at
		 FROM review_task WHERE submitted_by = $1 AND status = $2 ORDER BY submitted_at DESC`, submittedBy, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReviewTasks(rows)
}

func (r *ReviewTaskRepo) ExistsByNamespaceID(ctx context.Context, namespaceID int64) (bool, error) {
	var exists bool
	err := r.queryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM review_task WHERE namespace_id = $1)`, namespaceID,
	).Scan(&exists)
	return exists, err
}

func (r *ReviewTaskRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.exec(ctx, `DELETE FROM review_task WHERE id = $1`, id)
	return err
}

func (r *ReviewTaskRepo) DeleteByVersionIDs(ctx context.Context, versionIDs []int64) error {
	_, err := r.exec(ctx, `DELETE FROM review_task WHERE skill_version_id = ANY($1)`, versionIDs)
	return err
}

func (r *ReviewTaskRepo) UpdateStatusWithVersion(ctx context.Context, id int64, status string, reviewedBy string, reviewComment string, expectedVersion int) (int, error) {
	now := time.Now()
	tag, err := r.exec(ctx,
		`UPDATE review_task SET status = $2, reviewed_by = $3, review_comment = $4, reviewed_at = $5, version = version + 1
		 WHERE id = $1 AND version = $6`,
		id, status, reviewedBy, reviewComment, now, expectedVersion)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// PromotionRequestRepo implements review.PromotionRequestRepository.
type PromotionRequestRepo struct{ *DB }

// Compile-time assertion.
var _ review.PromotionRequestRepository = (*PromotionRequestRepo)(nil)

func NewPromotionRequestRepo(db *DB) *PromotionRequestRepo { return &PromotionRequestRepo{DB: db} }

func (r *PromotionRequestRepo) Save(ctx context.Context, req review.PromotionRequest) (review.PromotionRequest, error) {
	if req.SubmittedAt.IsZero() {
		req.SubmittedAt = time.Now()
	}

	if req.ID == 0 {
		err := r.queryRow(ctx,
			`INSERT INTO promotion_request (source_skill_id, source_version_id, target_namespace_id, target_skill_id, status, version, submitted_by, reviewed_by, review_comment, submitted_at, reviewed_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
			 RETURNING id, source_skill_id, source_version_id, target_namespace_id, target_skill_id, status, version, submitted_by, reviewed_by, review_comment, submitted_at, reviewed_at`,
			req.SourceSkillID, req.SourceVersionID, req.TargetNamespaceID, req.TargetSkillID, req.Status, req.Version,
			req.SubmittedBy, req.ReviewedBy, req.ReviewComment, req.SubmittedAt, req.ReviewedAt,
		).Scan(&req.ID, &req.SourceSkillID, &req.SourceVersionID, &req.TargetNamespaceID, &req.TargetSkillID,
			&req.Status, &req.Version, &req.SubmittedBy, &req.ReviewedBy, &req.ReviewComment, &req.SubmittedAt, &req.ReviewedAt)
		if err != nil {
			return review.PromotionRequest{}, err
		}
		return req, nil
	}

	// ID != 0: UPDATE existing row instead of inserting a duplicate.
	// Used by ApprovePromotion to write back target_skill_id.
	err := r.queryRow(ctx,
		`UPDATE promotion_request SET source_skill_id = $2, source_version_id = $3, target_namespace_id = $4,
		   target_skill_id = $5, status = $6, version = $7, submitted_by = $8, reviewed_by = $9,
		   review_comment = $10, submitted_at = $11, reviewed_at = $12
		 WHERE id = $1
		 RETURNING id, source_skill_id, source_version_id, target_namespace_id, target_skill_id, status, version, submitted_by, reviewed_by, review_comment, submitted_at, reviewed_at`,
		req.ID, req.SourceSkillID, req.SourceVersionID, req.TargetNamespaceID, req.TargetSkillID, req.Status, req.Version,
		req.SubmittedBy, req.ReviewedBy, req.ReviewComment, req.SubmittedAt, req.ReviewedAt,
	).Scan(&req.ID, &req.SourceSkillID, &req.SourceVersionID, &req.TargetNamespaceID, &req.TargetSkillID,
		&req.Status, &req.Version, &req.SubmittedBy, &req.ReviewedBy, &req.ReviewComment, &req.SubmittedAt, &req.ReviewedAt)
	if err != nil {
		return review.PromotionRequest{}, err
	}
	return req, nil
}

func (r *PromotionRequestRepo) FindByID(ctx context.Context, id int64) (*review.PromotionRequest, error) {
	var req review.PromotionRequest
	err := r.queryRow(ctx,
		`SELECT id, source_skill_id, source_version_id, target_namespace_id, target_skill_id, status, version, submitted_by, reviewed_by, review_comment, submitted_at, reviewed_at
		 FROM promotion_request WHERE id = $1`, id,
	).Scan(&req.ID, &req.SourceSkillID, &req.SourceVersionID, &req.TargetNamespaceID, &req.TargetSkillID,
		&req.Status, &req.Version, &req.SubmittedBy, &req.ReviewedBy, &req.ReviewComment, &req.SubmittedAt, &req.ReviewedAt)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *PromotionRequestRepo) FindBySourceVersionIDAndStatus(ctx context.Context, versionID int64, status string) (*review.PromotionRequest, error) {
	var req review.PromotionRequest
	err := r.queryRow(ctx,
		`SELECT id, source_skill_id, source_version_id, target_namespace_id, target_skill_id, status, version, submitted_by, reviewed_by, review_comment, submitted_at, reviewed_at
		 FROM promotion_request WHERE source_version_id = $1 AND status = $2`, versionID, status,
	).Scan(&req.ID, &req.SourceSkillID, &req.SourceVersionID, &req.TargetNamespaceID, &req.TargetSkillID,
		&req.Status, &req.Version, &req.SubmittedBy, &req.ReviewedBy, &req.ReviewComment, &req.SubmittedAt, &req.ReviewedAt)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *PromotionRequestRepo) FindBySourceSkillIDAndStatus(ctx context.Context, skillID int64, status string) (*review.PromotionRequest, error) {
	var req review.PromotionRequest
	err := r.queryRow(ctx,
		`SELECT id, source_skill_id, source_version_id, target_namespace_id, target_skill_id, status, version, submitted_by, reviewed_by, review_comment, submitted_at, reviewed_at
		 FROM promotion_request WHERE source_skill_id = $1 AND status = $2`, skillID, status,
	).Scan(&req.ID, &req.SourceSkillID, &req.SourceVersionID, &req.TargetNamespaceID, &req.TargetSkillID,
		&req.Status, &req.Version, &req.SubmittedBy, &req.ReviewedBy, &req.ReviewComment, &req.SubmittedAt, &req.ReviewedAt)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *PromotionRequestRepo) FindByStatus(ctx context.Context, status string) ([]review.PromotionRequest, error) {
	rows, err := r.query(ctx,
		`SELECT id, source_skill_id, source_version_id, target_namespace_id, target_skill_id, status, version, submitted_by, reviewed_by, review_comment, submitted_at, reviewed_at
		 FROM promotion_request WHERE status = $1 ORDER BY submitted_at DESC`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPromotionRequests(rows)
}

func (r *PromotionRequestRepo) FindByStatusPaged(ctx context.Context, status string, page int, size int) ([]review.PromotionRequest, bool, error) {
	rows, err := r.query(ctx,
		`SELECT id, source_skill_id, source_version_id, target_namespace_id, target_skill_id, status, version, submitted_by, reviewed_by, review_comment, submitted_at, reviewed_at
		 FROM promotion_request WHERE status = $1 ORDER BY submitted_at DESC, id DESC LIMIT $2 OFFSET $3`,
		status, size+1, page*size)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	reqs, err := scanPromotionRequests(rows)
	if err != nil {
		return nil, false, err
	}
	if len(reqs) > size {
		return reqs[:size], true, nil
	}
	return reqs, false, nil
}

func (r *PromotionRequestRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.exec(ctx, `DELETE FROM promotion_request WHERE id = $1`, id)
	return err
}

func (r *PromotionRequestRepo) ExistsByTargetNamespaceID(ctx context.Context, namespaceID int64) (bool, error) {
	var exists bool
	err := r.queryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM promotion_request WHERE target_namespace_id = $1)`, namespaceID,
	).Scan(&exists)
	return exists, err
}

func (r *PromotionRequestRepo) DeleteBySourceOrTargetSkillID(ctx context.Context, skillID int64) error {
	_, err := r.exec(ctx,
		`DELETE FROM promotion_request WHERE source_skill_id = $1 OR target_skill_id = $1`, skillID)
	return err
}

func (r *PromotionRequestRepo) UpdateStatusWithVersion(ctx context.Context, id int64, status string, reviewedBy string, reviewComment string, targetSkillID *int64, expectedVersion int) (int, error) {
	now := time.Now()
	tag, err := r.exec(ctx,
		`UPDATE promotion_request SET status = $2, reviewed_by = $3, review_comment = $4, target_skill_id = $5, reviewed_at = $6, version = version + 1
		 WHERE id = $1 AND version = $7`,
		id, status, reviewedBy, reviewComment, targetSkillID, now, expectedVersion)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// scanReviewTasks scans rows into a slice of ReviewTask.
func scanReviewTasks(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}) ([]review.ReviewTask, error) {
	var tasks []review.ReviewTask
	for rows.Next() {
		var t review.ReviewTask
		if err := rows.Scan(&t.ID, &t.SkillVersionID, &t.NamespaceID, &t.Status, &t.Version,
			&t.SubmittedBy, &t.ReviewedBy, &t.ReviewComment, &t.SubmittedAt, &t.ReviewedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// scanPromotionRequests scans rows into a slice of PromotionRequest.
func scanPromotionRequests(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}) ([]review.PromotionRequest, error) {
	var reqs []review.PromotionRequest
	for rows.Next() {
		var req review.PromotionRequest
		if err := rows.Scan(&req.ID, &req.SourceSkillID, &req.SourceVersionID, &req.TargetNamespaceID, &req.TargetSkillID,
			&req.Status, &req.Version, &req.SubmittedBy, &req.ReviewedBy, &req.ReviewComment, &req.SubmittedAt, &req.ReviewedAt); err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	return reqs, rows.Err()
}
