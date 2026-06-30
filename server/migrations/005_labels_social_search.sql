-- Migration 005: Labels, social actions, skill reports, security audit.
-- Collapses source V10, V34, V35, V40.
-- All timestamps use TIMESTAMPTZ from the start.

-- =============================================================================
-- Label definitions
-- =============================================================================
CREATE TABLE label_definition (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(64) NOT NULL UNIQUE,
    type VARCHAR(16) NOT NULL CHECK (type IN ('RECOMMENDED', 'PRIVILEGED')),
    visible_in_filter BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_by VARCHAR(128) REFERENCES user_account(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_label_definition_visible_sort
    ON label_definition(visible_in_filter, type, sort_order, id);

-- =============================================================================
-- Label translations
-- =============================================================================
CREATE TABLE label_translation (
    id BIGSERIAL PRIMARY KEY,
    label_id BIGINT NOT NULL REFERENCES label_definition(id) ON DELETE CASCADE,
    locale VARCHAR(16) NOT NULL,
    display_name VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(label_id, locale)
);
CREATE INDEX idx_label_translation_label_id ON label_translation(label_id);

-- =============================================================================
-- Skill-label assignments
-- =============================================================================
CREATE TABLE skill_label (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL REFERENCES skill(id) ON DELETE CASCADE,
    label_id BIGINT NOT NULL REFERENCES label_definition(id) ON DELETE CASCADE,
    created_by VARCHAR(128) REFERENCES user_account(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(skill_id, label_id)
);
CREATE INDEX idx_skill_label_label_id ON skill_label(label_id);
CREATE INDEX idx_skill_label_skill_id ON skill_label(skill_id);

-- =============================================================================
-- Skill stars
-- =============================================================================
CREATE TABLE skill_star (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL REFERENCES skill(id),
    user_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(skill_id, user_id)
);
CREATE INDEX idx_skill_star_user_id ON skill_star(user_id);
CREATE INDEX idx_skill_star_skill_id ON skill_star(skill_id);

-- =============================================================================
-- Skill ratings (1-5)
-- =============================================================================
CREATE TABLE skill_rating (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL REFERENCES skill(id),
    user_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    score SMALLINT NOT NULL CHECK (score >= 1 AND score <= 5),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(skill_id, user_id)
);
CREATE INDEX idx_skill_rating_skill_id ON skill_rating(skill_id);

-- =============================================================================
-- Skill subscriptions
-- =============================================================================
CREATE TABLE skill_subscription (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL REFERENCES skill(id) ON DELETE CASCADE,
    user_id VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(skill_id, user_id)
);
CREATE INDEX idx_skill_subscription_user ON skill_subscription(user_id, created_at DESC);
CREATE INDEX idx_skill_subscription_skill ON skill_subscription(skill_id);

-- =============================================================================
-- Skill reports
-- =============================================================================
CREATE TABLE skill_report (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL REFERENCES skill(id) ON DELETE CASCADE,
    namespace_id BIGINT NOT NULL REFERENCES namespace(id) ON DELETE CASCADE,
    reporter_id VARCHAR(128) NOT NULL,
    reason VARCHAR(200) NOT NULL,
    details TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    handled_by VARCHAR(128),
    handle_comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    handled_at TIMESTAMPTZ
);
CREATE INDEX idx_skill_report_status_created_at ON skill_report(status, created_at DESC);
CREATE INDEX idx_skill_report_skill_id ON skill_report(skill_id);

-- =============================================================================
-- Security audit (soft-delete via deleted_at)
-- =============================================================================
CREATE TABLE security_audit (
    id BIGSERIAL PRIMARY KEY,
    skill_version_id BIGINT NOT NULL,
    scan_id VARCHAR(100),
    scanner_type VARCHAR(50) NOT NULL DEFAULT 'skill-scanner',
    verdict VARCHAR(20) NOT NULL,
    is_safe BOOLEAN NOT NULL,
    max_severity VARCHAR(20),
    findings_count INT NOT NULL DEFAULT 0,
    findings JSONB NOT NULL DEFAULT '[]'::jsonb,
    scan_duration_seconds DOUBLE PRECISION,
    scanned_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ DEFAULT NULL
);
CREATE INDEX idx_security_audit_version_active
    ON security_audit(skill_version_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_security_audit_verdict ON security_audit(verdict);
CREATE INDEX idx_security_audit_version_type_latest
    ON security_audit(skill_version_id, scanner_type, created_at DESC) WHERE deleted_at IS NULL;
