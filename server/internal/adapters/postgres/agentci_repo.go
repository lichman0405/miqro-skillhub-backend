package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"miqro-skillhub/server/sdk/skillhub/agentci"
)

// ── AgentWorkerRepo ─────────────────────────────────────────────────────────

type AgentWorkerRepo struct{ DB *DB }

func NewAgentWorkerRepo(db *DB) *AgentWorkerRepo { return &AgentWorkerRepo{DB: db} }

func (r *AgentWorkerRepo) Create(ctx context.Context, w agentci.AgentWorker) (agentci.AgentWorker, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO agent_workers (name, type, config_json, status, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		w.Name, w.Type, w.ConfigJSON, w.Status, w.CreatedAt, w.UpdatedAt,
	).Scan(&w.ID)
	return w, err
}

func (r *AgentWorkerRepo) FindByID(ctx context.Context, id int64) (*agentci.AgentWorker, error) {
	var w agentci.AgentWorker
	err := r.DB.queryRow(ctx,
		`SELECT id, name, type, config_json, status, created_at, updated_at
		 FROM agent_workers WHERE id=$1`, id,
	).Scan(&w.ID, &w.Name, &w.Type, &w.ConfigJSON, &w.Status, &w.CreatedAt, &w.UpdatedAt)
	if noRows(err) {
		return nil, nil
	}
	return &w, err
}

func (r *AgentWorkerRepo) FindByType(ctx context.Context, workerType string) ([]agentci.AgentWorker, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, name, type, config_json, status, created_at, updated_at
		 FROM agent_workers WHERE type=$1 AND status='ACTIVE'`, workerType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []agentci.AgentWorker
	for rows.Next() {
		var w agentci.AgentWorker
		if err := rows.Scan(&w.ID, &w.Name, &w.Type, &w.ConfigJSON, &w.Status, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, nil
}

func (r *AgentWorkerRepo) List(ctx context.Context) ([]agentci.AgentWorker, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, name, type, config_json, status, created_at, updated_at
		 FROM agent_workers ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []agentci.AgentWorker
	for rows.Next() {
		var w agentci.AgentWorker
		if err := rows.Scan(&w.ID, &w.Name, &w.Type, &w.ConfigJSON, &w.Status, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, nil
}

func (r *AgentWorkerRepo) Update(ctx context.Context, w agentci.AgentWorker) (agentci.AgentWorker, error) {
	_, err := r.DB.exec(ctx,
		`UPDATE agent_workers SET name=$2, type=$3, config_json=$4, status=$5, updated_at=$6 WHERE id=$1`,
		w.ID, w.Name, w.Type, w.ConfigJSON, w.Status, w.UpdatedAt)
	return w, err
}

func (r *AgentWorkerRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.exec(ctx, `DELETE FROM agent_workers WHERE id=$1`, id)
	return err
}

// ── PipelineDefinitionRepo ──────────────────────────────────────────────────

type PipelineDefinitionRepo struct{ DB *DB }

func NewPipelineDefinitionRepo(db *DB) *PipelineDefinitionRepo {
	return &PipelineDefinitionRepo{DB: db}
}

func (r *PipelineDefinitionRepo) Create(ctx context.Context, p agentci.PipelineDefinition) (agentci.PipelineDefinition, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO ci_pipeline_definitions (name, description, trigger_on, steps_json, enabled, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		p.Name, p.Description, p.TriggerOn, p.StepsJSON, p.Enabled, p.CreatedAt, p.UpdatedAt,
	).Scan(&p.ID)
	return p, err
}

func (r *PipelineDefinitionRepo) FindByID(ctx context.Context, id int64) (*agentci.PipelineDefinition, error) {
	var p agentci.PipelineDefinition
	err := r.DB.queryRow(ctx,
		`SELECT id, name, description, trigger_on, steps_json, enabled, created_at, updated_at
		 FROM ci_pipeline_definitions WHERE id=$1`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.TriggerOn, &p.StepsJSON, &p.Enabled, &p.CreatedAt, &p.UpdatedAt)
	if noRows(err) {
		return nil, nil
	}
	return &p, err
}

func (r *PipelineDefinitionRepo) FindByTriggerOn(ctx context.Context, triggerType string) ([]agentci.PipelineDefinition, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, name, description, trigger_on, steps_json, enabled, created_at, updated_at
		 FROM ci_pipeline_definitions WHERE trigger_on LIKE '%' || $1 || '%' ORDER BY id`, triggerType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []agentci.PipelineDefinition
	for rows.Next() {
		var p agentci.PipelineDefinition
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.TriggerOn, &p.StepsJSON, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *PipelineDefinitionRepo) List(ctx context.Context) ([]agentci.PipelineDefinition, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, name, description, trigger_on, steps_json, enabled, created_at, updated_at
		 FROM ci_pipeline_definitions ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []agentci.PipelineDefinition
	for rows.Next() {
		var p agentci.PipelineDefinition
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.TriggerOn, &p.StepsJSON, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *PipelineDefinitionRepo) Update(ctx context.Context, p agentci.PipelineDefinition) (agentci.PipelineDefinition, error) {
	_, err := r.DB.exec(ctx,
		`UPDATE ci_pipeline_definitions SET name=$2, description=$3, trigger_on=$4, steps_json=$5, enabled=$6, updated_at=$7 WHERE id=$1`,
		p.ID, p.Name, p.Description, p.TriggerOn, p.StepsJSON, p.Enabled, p.UpdatedAt)
	return p, err
}

func (r *PipelineDefinitionRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.exec(ctx, `DELETE FROM ci_pipeline_definitions WHERE id=$1`, id)
	return err
}

// ── PipelineRunRepo ─────────────────────────────────────────────────────────

type PipelineRunRepo struct{ DB *DB }

func NewPipelineRunRepo(db *DB) *PipelineRunRepo { return &PipelineRunRepo{DB: db} }

func (r *PipelineRunRepo) Create(ctx context.Context, pr agentci.PipelineRun) (agentci.PipelineRun, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO ci_pipeline_runs (pipeline_id, skill_id, version_id, release_id, trigger_type, triggered_by,
		 status, check_count, passed_count, failed_count, skipped_count, started_at, completed_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING id`,
		pr.PipelineID, pr.SkillID, pr.VersionID, pr.ReleaseID, pr.TriggerType, pr.TriggeredBy,
		pr.Status, pr.CheckCount, pr.PassedCount, pr.FailedCount, pr.SkippedCount,
		pr.StartedAt, pr.CompletedAt, pr.CreatedAt, pr.UpdatedAt,
	).Scan(&pr.ID)
	return pr, err
}

func (r *PipelineRunRepo) FindByID(ctx context.Context, id int64) (*agentci.PipelineRun, error) {
	var pr agentci.PipelineRun
	err := r.DB.queryRow(ctx,
		`SELECT id, pipeline_id, skill_id, version_id, release_id, trigger_type, triggered_by,
		 status, check_count, passed_count, failed_count, skipped_count, started_at, completed_at, created_at, updated_at
		 FROM ci_pipeline_runs WHERE id=$1`, id,
	).Scan(&pr.ID, &pr.PipelineID, &pr.SkillID, &pr.VersionID, &pr.ReleaseID, &pr.TriggerType, &pr.TriggeredBy,
		&pr.Status, &pr.CheckCount, &pr.PassedCount, &pr.FailedCount, &pr.SkippedCount,
		&pr.StartedAt, &pr.CompletedAt, &pr.CreatedAt, &pr.UpdatedAt)
	if noRows(err) {
		return nil, nil
	}
	return &pr, err
}

func (r *PipelineRunRepo) FindBySkillID(ctx context.Context, skillID int64, offset, limit int) ([]agentci.PipelineRun, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, pipeline_id, skill_id, version_id, release_id, trigger_type, triggered_by,
		 status, check_count, passed_count, failed_count, skipped_count, started_at, completed_at, created_at, updated_at
		 FROM ci_pipeline_runs WHERE skill_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		skillID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPipelineRuns(rows)
}

func (r *PipelineRunRepo) CountBySkillID(ctx context.Context, skillID int64) (int64, error) {
	var count int64
	err := r.DB.queryRow(ctx,
		`SELECT COUNT(*) FROM ci_pipeline_runs WHERE skill_id=$1`, skillID).Scan(&count)
	return count, err
}

func (r *PipelineRunRepo) FindByVersionID(ctx context.Context, versionID int64) ([]agentci.PipelineRun, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, pipeline_id, skill_id, version_id, release_id, trigger_type, triggered_by,
		 status, check_count, passed_count, failed_count, skipped_count, started_at, completed_at, created_at, updated_at
		 FROM ci_pipeline_runs WHERE version_id=$1 ORDER BY created_at DESC`, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPipelineRuns(rows)
}

func (r *PipelineRunRepo) FindByReleaseID(ctx context.Context, releaseID int64) ([]agentci.PipelineRun, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, pipeline_id, skill_id, version_id, release_id, trigger_type, triggered_by,
		 status, check_count, passed_count, failed_count, skipped_count, started_at, completed_at, created_at, updated_at
		 FROM ci_pipeline_runs WHERE release_id=$1 ORDER BY created_at DESC`, releaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPipelineRuns(rows)
}

func (r *PipelineRunRepo) FindPending(ctx context.Context, limit int) ([]agentci.PipelineRun, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, pipeline_id, skill_id, version_id, release_id, trigger_type, triggered_by,
		 status, check_count, passed_count, failed_count, skipped_count, started_at, completed_at, created_at, updated_at
		 FROM ci_pipeline_runs WHERE status IN ('PENDING','RUNNING') ORDER BY created_at ASC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPipelineRuns(rows)
}

func (r *PipelineRunRepo) ClaimPending(ctx context.Context, id int64) (*agentci.PipelineRun, error) {
	tag, err := r.DB.exec(ctx,
		`UPDATE ci_pipeline_runs SET status='RUNNING', started_at=NOW(), updated_at=NOW()
		 WHERE id=$1 AND status='PENDING'`, id)
	if err != nil {
		return nil, err
	}
	rowsAffected := tag.RowsAffected()
	if rowsAffected == 0 {
		return nil, nil // already claimed by another worker
	}
	return r.FindByID(ctx, id)
}

func (r *PipelineRunRepo) Update(ctx context.Context, pr agentci.PipelineRun) (agentci.PipelineRun, error) {
	_, err := r.DB.exec(ctx,
		`UPDATE ci_pipeline_runs SET status=$2, check_count=$3, passed_count=$4, failed_count=$5,
		 skipped_count=$6, started_at=$7, completed_at=$8, updated_at=$9 WHERE id=$1`,
		pr.ID, pr.Status, pr.CheckCount, pr.PassedCount, pr.FailedCount,
		pr.SkippedCount, pr.StartedAt, pr.CompletedAt, pr.UpdatedAt)
	return pr, err
}

// ── CheckRunRepo ────────────────────────────────────────────────────────────

type CheckRunRepo struct{ DB *DB }

func NewCheckRunRepo(db *DB) *CheckRunRepo { return &CheckRunRepo{DB: db} }

func (r *CheckRunRepo) Create(ctx context.Context, cr agentci.CheckRun) (agentci.CheckRun, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO ci_check_runs (pipeline_run_id, skill_id, version_id, release_id, name, runner_type,
		 status, conclusion, summary, is_blocking, started_at, completed_at, duration_ms, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING id`,
		cr.PipelineRunID, cr.SkillID, cr.VersionID, cr.ReleaseID, cr.Name, cr.RunnerType,
		cr.Status, cr.Conclusion, cr.Summary, cr.IsBlocking, cr.StartedAt, cr.CompletedAt,
		cr.DurationMs, cr.CreatedAt, cr.UpdatedAt,
	).Scan(&cr.ID)
	return cr, err
}

func (r *CheckRunRepo) FindByID(ctx context.Context, id int64) (*agentci.CheckRun, error) {
	var cr agentci.CheckRun
	err := r.DB.queryRow(ctx,
		`SELECT id, pipeline_run_id, skill_id, version_id, release_id, name, runner_type,
		 status, conclusion, summary, is_blocking, started_at, completed_at, duration_ms, created_at, updated_at
		 FROM ci_check_runs WHERE id=$1`, id,
	).Scan(&cr.ID, &cr.PipelineRunID, &cr.SkillID, &cr.VersionID, &cr.ReleaseID, &cr.Name, &cr.RunnerType,
		&cr.Status, &cr.Conclusion, &cr.Summary, &cr.IsBlocking, &cr.StartedAt, &cr.CompletedAt,
		&cr.DurationMs, &cr.CreatedAt, &cr.UpdatedAt)
	if noRows(err) {
		return nil, nil
	}
	return &cr, err
}

func (r *CheckRunRepo) FindByPipelineRunID(ctx context.Context, pipelineRunID int64) ([]agentci.CheckRun, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, pipeline_run_id, skill_id, version_id, release_id, name, runner_type,
		 status, conclusion, summary, is_blocking, started_at, completed_at, duration_ms, created_at, updated_at
		 FROM ci_check_runs WHERE pipeline_run_id=$1 ORDER BY id`, pipelineRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCheckRuns(rows)
}

func (r *CheckRunRepo) FindBySkillID(ctx context.Context, skillID int64, offset, limit int) ([]agentci.CheckRun, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, pipeline_run_id, skill_id, version_id, release_id, name, runner_type,
		 status, conclusion, summary, is_blocking, started_at, completed_at, duration_ms, created_at, updated_at
		 FROM ci_check_runs WHERE skill_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		skillID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCheckRuns(rows)
}

func (r *CheckRunRepo) CountBySkillID(ctx context.Context, skillID int64) (int64, error) {
	var count int64
	err := r.DB.queryRow(ctx,
		`SELECT COUNT(*) FROM ci_check_runs WHERE skill_id=$1`, skillID).Scan(&count)
	return count, err
}

func (r *CheckRunRepo) FindByVersionID(ctx context.Context, versionID int64) ([]agentci.CheckRun, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, pipeline_run_id, skill_id, version_id, release_id, name, runner_type,
		 status, conclusion, summary, is_blocking, started_at, completed_at, duration_ms, created_at, updated_at
		 FROM ci_check_runs WHERE version_id=$1 ORDER BY created_at DESC`, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCheckRuns(rows)
}

func (r *CheckRunRepo) FindByReleaseID(ctx context.Context, releaseID int64) ([]agentci.CheckRun, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, pipeline_run_id, skill_id, version_id, release_id, name, runner_type,
		 status, conclusion, summary, is_blocking, started_at, completed_at, duration_ms, created_at, updated_at
		 FROM ci_check_runs WHERE release_id=$1 ORDER BY created_at DESC`, releaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCheckRuns(rows)
}

func (r *CheckRunRepo) Update(ctx context.Context, cr agentci.CheckRun) (agentci.CheckRun, error) {
	_, err := r.DB.exec(ctx,
		`UPDATE ci_check_runs SET status=$2, conclusion=$3, summary=$4, is_blocking=$5,
		 started_at=$6, completed_at=$7, duration_ms=$8, updated_at=$9 WHERE id=$1`,
		cr.ID, cr.Status, cr.Conclusion, cr.Summary, cr.IsBlocking,
		cr.StartedAt, cr.CompletedAt, cr.DurationMs, cr.UpdatedAt)
	return cr, err
}

// ── CheckStepRepo ───────────────────────────────────────────────────────────

type CheckStepRepo struct{ DB *DB }

func NewCheckStepRepo(db *DB) *CheckStepRepo { return &CheckStepRepo{DB: db} }

func (r *CheckStepRepo) Create(ctx context.Context, s agentci.CheckStep) (agentci.CheckStep, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO ci_check_steps (check_run_id, name, status, started_at, completed_at, duration_ms, log_ref, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
		s.CheckRunID, s.Name, s.Status, s.StartedAt, s.CompletedAt, s.DurationMs, s.LogRef, s.CreatedAt,
	).Scan(&s.ID)
	return s, err
}

func (r *CheckStepRepo) FindByCheckRunID(ctx context.Context, checkRunID int64) ([]agentci.CheckStep, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, check_run_id, name, status, started_at, completed_at, duration_ms, log_ref, created_at
		 FROM ci_check_steps WHERE check_run_id=$1 ORDER BY id`, checkRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []agentci.CheckStep
	for rows.Next() {
		var s agentci.CheckStep
		if err := rows.Scan(&s.ID, &s.CheckRunID, &s.Name, &s.Status, &s.StartedAt, &s.CompletedAt, &s.DurationMs, &s.LogRef, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (r *CheckStepRepo) Update(ctx context.Context, s agentci.CheckStep) (agentci.CheckStep, error) {
	_, err := r.DB.exec(ctx,
		`UPDATE ci_check_steps SET status=$2, started_at=$3, completed_at=$4, duration_ms=$5, log_ref=$6 WHERE id=$1`,
		s.ID, s.Status, s.StartedAt, s.CompletedAt, s.DurationMs, s.LogRef)
	return s, err
}

// ── CheckArtifactRepo ───────────────────────────────────────────────────────

type CheckArtifactRepo struct{ DB *DB }

func NewCheckArtifactRepo(db *DB) *CheckArtifactRepo { return &CheckArtifactRepo{DB: db} }

func (r *CheckArtifactRepo) Create(ctx context.Context, a agentci.CheckArtifact) (agentci.CheckArtifact, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO ci_check_artifacts (check_run_id, name, content_type, size, storage_key, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		a.CheckRunID, a.Name, a.ContentType, a.Size, a.StorageKey, a.CreatedAt,
	).Scan(&a.ID)
	return a, err
}

func (r *CheckArtifactRepo) FindByCheckRunID(ctx context.Context, checkRunID int64) ([]agentci.CheckArtifact, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, check_run_id, name, content_type, size, storage_key, created_at
		 FROM ci_check_artifacts WHERE check_run_id=$1 ORDER BY id`, checkRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []agentci.CheckArtifact
	for rows.Next() {
		var a agentci.CheckArtifact
		if err := rows.Scan(&a.ID, &a.CheckRunID, &a.Name, &a.ContentType, &a.Size, &a.StorageKey, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func (r *CheckArtifactRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.exec(ctx, `DELETE FROM ci_check_artifacts WHERE id=$1`, id)
	return err
}

// ── GatePolicyRepo ──────────────────────────────────────────────────────────

type GatePolicyRepo struct{ DB *DB }

func NewGatePolicyRepo(db *DB) *GatePolicyRepo { return &GatePolicyRepo{DB: db} }

func (r *GatePolicyRepo) Create(ctx context.Context, g agentci.GatePolicy) (agentci.GatePolicy, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO ci_gate_policies (name, description, trigger_on, required_rule, enabled, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		g.Name, g.Description, g.TriggerOn, g.RequiredRule, g.Enabled, g.CreatedAt, g.UpdatedAt,
	).Scan(&g.ID)
	return g, err
}

func (r *GatePolicyRepo) FindByID(ctx context.Context, id int64) (*agentci.GatePolicy, error) {
	var g agentci.GatePolicy
	err := r.DB.queryRow(ctx,
		`SELECT id, name, description, trigger_on, required_rule, enabled, created_at, updated_at
		 FROM ci_gate_policies WHERE id=$1`, id,
	).Scan(&g.ID, &g.Name, &g.Description, &g.TriggerOn, &g.RequiredRule, &g.Enabled, &g.CreatedAt, &g.UpdatedAt)
	if noRows(err) {
		return nil, nil
	}
	return &g, err
}

func (r *GatePolicyRepo) FindByTriggerOn(ctx context.Context, triggerType string) ([]agentci.GatePolicy, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, name, description, trigger_on, required_rule, enabled, created_at, updated_at
		 FROM ci_gate_policies WHERE trigger_on=$1 AND enabled=TRUE ORDER BY id`, triggerType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []agentci.GatePolicy
	for rows.Next() {
		var g agentci.GatePolicy
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.TriggerOn, &g.RequiredRule, &g.Enabled, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

func (r *GatePolicyRepo) List(ctx context.Context) ([]agentci.GatePolicy, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, name, description, trigger_on, required_rule, enabled, created_at, updated_at
		 FROM ci_gate_policies ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []agentci.GatePolicy
	for rows.Next() {
		var g agentci.GatePolicy
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.TriggerOn, &g.RequiredRule, &g.Enabled, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

func (r *GatePolicyRepo) Update(ctx context.Context, g agentci.GatePolicy) (agentci.GatePolicy, error) {
	_, err := r.DB.exec(ctx,
		`UPDATE ci_gate_policies SET name=$2, description=$3, trigger_on=$4, required_rule=$5, enabled=$6, updated_at=$7 WHERE id=$1`,
		g.ID, g.Name, g.Description, g.TriggerOn, g.RequiredRule, g.Enabled, g.UpdatedAt)
	return g, err
}

func (r *GatePolicyRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.exec(ctx, `DELETE FROM ci_gate_policies WHERE id=$1`, id)
	return err
}

// ── Scan helpers ─────────────────────────────────────────────────────────────

func scanPipelineRuns(rows pgx.Rows) ([]agentci.PipelineRun, error) {
	defer rows.Close()
	var out []agentci.PipelineRun
	for rows.Next() {
		var pr agentci.PipelineRun
		if err := rows.Scan(&pr.ID, &pr.PipelineID, &pr.SkillID, &pr.VersionID, &pr.ReleaseID, &pr.TriggerType,
			&pr.TriggeredBy, &pr.Status, &pr.CheckCount, &pr.PassedCount, &pr.FailedCount, &pr.SkippedCount,
			&pr.StartedAt, &pr.CompletedAt, &pr.CreatedAt, &pr.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, pr)
	}
	return out, nil
}

func scanCheckRuns(rows pgx.Rows) ([]agentci.CheckRun, error) {
	defer rows.Close()
	var out []agentci.CheckRun
	for rows.Next() {
		var cr agentci.CheckRun
		if err := rows.Scan(&cr.ID, &cr.PipelineRunID, &cr.SkillID, &cr.VersionID, &cr.ReleaseID, &cr.Name,
			&cr.RunnerType, &cr.Status, &cr.Conclusion, &cr.Summary, &cr.IsBlocking,
			&cr.StartedAt, &cr.CompletedAt, &cr.DurationMs, &cr.CreatedAt, &cr.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, cr)
	}
	return out, nil
}
