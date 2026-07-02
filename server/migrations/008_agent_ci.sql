-- Migration 008: Agent CI/CD subsystem.
-- Agent workers, pipeline definitions, pipeline runs, check runs, check steps,
-- check artifacts, and gate policies.
-- Mirrors the scanner architecture from SkillScannerConfig.java / SecurityAudit.java.
-- All timestamps use TIMESTAMPTZ.

-- =============================================================================
-- Agent workers
-- =============================================================================
CREATE TABLE agent_workers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(256) NOT NULL,
    type VARCHAR(64) NOT NULL CHECK (type IN ('local', 'claude-code', 'codex', 'container')),
    config_json JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE', 'ERROR')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_agent_workers_type ON agent_workers(type, status);

-- =============================================================================
-- CI pipeline definitions
-- =============================================================================
CREATE TABLE ci_pipeline_definitions (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(256) NOT NULL,
    description TEXT,
    trigger_on VARCHAR(256) NOT NULL,
    steps_json JSONB NOT NULL DEFAULT '[]',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_ci_pipeline_trigger ON ci_pipeline_definitions(trigger_on, enabled);

-- =============================================================================
-- CI pipeline runs
-- =============================================================================
CREATE TABLE ci_pipeline_runs (
    id BIGSERIAL PRIMARY KEY,
    pipeline_id BIGINT NOT NULL REFERENCES ci_pipeline_definitions(id),
    skill_id BIGINT NOT NULL REFERENCES skill(id),
    version_id BIGINT REFERENCES skill_version(id),
    release_id BIGINT REFERENCES skill_release(id),
    trigger_type VARCHAR(32) NOT NULL CHECK (trigger_type IN ('publish', 'review', 'release', 'manual')),
    triggered_by VARCHAR(128) NOT NULL REFERENCES user_account(id),
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'RUNNING', 'COMPLETED', 'FAILED', 'CANCELLED')),
    check_count INT NOT NULL DEFAULT 0,
    passed_count INT NOT NULL DEFAULT 0,
    failed_count INT NOT NULL DEFAULT 0,
    skipped_count INT NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_ci_pipeline_run_skill ON ci_pipeline_runs(skill_id, created_at DESC);
CREATE INDEX idx_ci_pipeline_run_version ON ci_pipeline_runs(version_id);
CREATE INDEX idx_ci_pipeline_run_release ON ci_pipeline_runs(release_id);
CREATE INDEX idx_ci_pipeline_run_status ON ci_pipeline_runs(status, created_at DESC);

-- =============================================================================
-- CI check runs
-- =============================================================================
CREATE TABLE ci_check_runs (
    id BIGSERIAL PRIMARY KEY,
    pipeline_run_id BIGINT NOT NULL REFERENCES ci_pipeline_runs(id) ON DELETE CASCADE,
    skill_id BIGINT NOT NULL REFERENCES skill(id),
    version_id BIGINT REFERENCES skill_version(id),
    release_id BIGINT REFERENCES skill_release(id),
    name VARCHAR(256) NOT NULL,
    runner_type VARCHAR(64) NOT NULL DEFAULT 'deterministic',
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'RUNNING', 'PASSED', 'FAILED', 'ERROR', 'SKIPPED', 'CANCELLED')),
    conclusion TEXT,
    summary TEXT,
    is_blocking BOOLEAN NOT NULL DEFAULT TRUE,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_ci_check_run_pipeline ON ci_check_runs(pipeline_run_id);
CREATE INDEX idx_ci_check_run_skill ON ci_check_runs(skill_id, created_at DESC);
CREATE INDEX idx_ci_check_run_version ON ci_check_runs(version_id);
CREATE INDEX idx_ci_check_run_release ON ci_check_runs(release_id);

-- =============================================================================
-- CI check steps
-- =============================================================================
CREATE TABLE ci_check_steps (
    id BIGSERIAL PRIMARY KEY,
    check_run_id BIGINT NOT NULL REFERENCES ci_check_runs(id) ON DELETE CASCADE,
    name VARCHAR(256) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'RUNNING', 'PASSED', 'FAILED', 'ERROR', 'SKIPPED', 'CANCELLED')),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms BIGINT,
    log_ref VARCHAR(1024),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_ci_check_step_check ON ci_check_steps(check_run_id);

-- =============================================================================
-- CI check artifacts
-- =============================================================================
CREATE TABLE ci_check_artifacts (
    id BIGSERIAL PRIMARY KEY,
    check_run_id BIGINT NOT NULL REFERENCES ci_check_runs(id) ON DELETE CASCADE,
    name VARCHAR(512) NOT NULL,
    content_type VARCHAR(256) NOT NULL DEFAULT 'application/octet-stream',
    size BIGINT NOT NULL DEFAULT 0,
    storage_key VARCHAR(1024) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_ci_check_artifact_check ON ci_check_artifacts(check_run_id);

-- =============================================================================
-- CI gate policies
-- =============================================================================
CREATE TABLE ci_gate_policies (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(256) NOT NULL,
    description TEXT,
    trigger_on VARCHAR(64) NOT NULL CHECK (trigger_on IN ('review_approve', 'release_publish')),
    required_rule VARCHAR(32) NOT NULL DEFAULT 'all_passed' CHECK (required_rule IN ('all_passed', 'required_passed', 'no_critical')),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_ci_gate_policy_trigger ON ci_gate_policies(trigger_on, enabled);

-- =============================================================================
-- Default gate policies
-- =============================================================================
INSERT INTO ci_gate_policies (name, description, trigger_on, required_rule, enabled)
VALUES ('Release Gate', 'All required checks must pass before release publication', 'release_publish', 'required_passed', TRUE);

INSERT INTO ci_gate_policies (name, description, trigger_on, required_rule, enabled)
VALUES ('Review Gate', 'No critical failures before review approval', 'review_approve', 'no_critical', TRUE);

-- =============================================================================
-- Default pipeline definition
-- =============================================================================
INSERT INTO ci_pipeline_definitions (name, description, trigger_on, steps_json, enabled)
VALUES ('Standard Publish Pipeline',
        'Default CI checks for skill publish and release events',
        'publish,review,release,manual',
        '[
          {"name": "manifest-validation", "runnerType": "deterministic"},
          {"name": "package-policy-validation", "runnerType": "deterministic"},
          {"name": "secret-scan", "runnerType": "deterministic"},
          {"name": "install-smoke-test", "runnerType": "deterministic"},
          {"name": "documentation-quality", "runnerType": "deterministic"},
          {"name": "release-notes-suggestion", "runnerType": "llm"}
        ]',
        TRUE);

-- =============================================================================
-- Default agent worker (local deterministic runner)
-- =============================================================================
INSERT INTO agent_workers (name, type, config_json, status)
VALUES ('local-deterministic-runner', 'local', '{}', 'ACTIVE');
