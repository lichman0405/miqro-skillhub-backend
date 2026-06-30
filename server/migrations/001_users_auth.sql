-- Migration 001: Users, auth, RBAC, audit log, profile changes.
-- Collapses source V1, V5, V8, V27, V39 into a single greenfield migration.
-- All timestamps use TIMESTAMPTZ from the start.

-- =============================================================================
-- User accounts
-- =============================================================================
CREATE TABLE user_account (
    id VARCHAR(128) PRIMARY KEY,
    display_name VARCHAR(128) NOT NULL,
    email VARCHAR(256),
    avatar_url VARCHAR(512),
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    merged_to_user_id VARCHAR(128),
    system_account BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_user_account_email ON user_account(email);
CREATE INDEX idx_user_account_status ON user_account(status);

-- =============================================================================
-- Identity bindings (OAuth / OIDC)
-- =============================================================================
CREATE TABLE identity_binding (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    provider_code VARCHAR(64) NOT NULL,
    subject VARCHAR(256) NOT NULL,
    login_name VARCHAR(128),
    extra_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider_code, subject)
);
CREATE INDEX idx_identity_binding_user_id ON identity_binding(user_id);

-- =============================================================================
-- Local credentials
-- =============================================================================
CREATE TABLE local_credential (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    username VARCHAR(64) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    failed_attempts INT NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX idx_local_credential_username ON local_credential(username);
CREATE UNIQUE INDEX idx_local_credential_user_id ON local_credential(user_id);

-- =============================================================================
-- API tokens (sk_ prefix, SHA-256 hash stored, raw token shown once)
-- =============================================================================
CREATE TABLE api_token (
    id BIGSERIAL PRIMARY KEY,
    subject_type VARCHAR(32) NOT NULL DEFAULT 'USER',
    subject_id VARCHAR(128) NOT NULL,
    user_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    name VARCHAR(64) NOT NULL,
    token_prefix VARCHAR(16) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    scope_json JSONB NOT NULL,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_api_token_user_id ON api_token(user_id);
CREATE INDEX idx_api_token_hash ON api_token(token_hash);
CREATE UNIQUE INDEX uk_api_token_user_active_name
    ON api_token(user_id, LOWER(name)) WHERE revoked_at IS NULL;

-- =============================================================================
-- RBAC: roles, permissions, role-permission bindings, user-role bindings
-- =============================================================================
CREATE TABLE role (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL,
    description VARCHAR(512),
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE permission (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(128) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL,
    group_code VARCHAR(64)
);

CREATE TABLE role_permission (
    role_id BIGINT NOT NULL REFERENCES role(id),
    permission_id BIGINT NOT NULL REFERENCES permission(id),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_role_binding (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    role_id BIGINT NOT NULL REFERENCES role(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, role_id)
);
CREATE INDEX idx_user_role_binding_user_id ON user_role_binding(user_id);

-- =============================================================================
-- Account merge
-- =============================================================================
CREATE TABLE account_merge_request (
    id BIGSERIAL PRIMARY KEY,
    primary_user_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    secondary_user_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    verification_token VARCHAR(255),
    token_expires_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_merge_primary_status ON account_merge_request(primary_user_id, status);
CREATE UNIQUE INDEX idx_merge_secondary_pending
    ON account_merge_request(secondary_user_id) WHERE status = 'PENDING';
CREATE INDEX idx_merge_token_pending
    ON account_merge_request(verification_token) WHERE status = 'PENDING';

-- =============================================================================
-- Password reset
-- =============================================================================
CREATE TABLE password_reset_request (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(128) NOT NULL REFERENCES user_account(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    code_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    requested_by_admin BOOLEAN NOT NULL DEFAULT FALSE,
    requested_by_user_id VARCHAR(128) REFERENCES user_account(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_password_reset_request_user_id ON password_reset_request(user_id);
CREATE INDEX idx_password_reset_request_expires_at ON password_reset_request(expires_at);

-- =============================================================================
-- Audit log
-- =============================================================================
CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    actor_user_id VARCHAR(128) REFERENCES user_account(id),
    action VARCHAR(64) NOT NULL,
    target_type VARCHAR(64),
    target_id BIGINT,
    request_id VARCHAR(64),
    client_ip VARCHAR(64),
    user_agent VARCHAR(512),
    detail_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_audit_log_actor ON audit_log(actor_user_id);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);
CREATE INDEX idx_audit_log_request_id ON audit_log(request_id);
CREATE INDEX idx_audit_log_actor_time ON audit_log(actor_user_id, created_at DESC);
CREATE INDEX idx_audit_log_action_time ON audit_log(action, created_at DESC);

-- =============================================================================
-- Profile change requests
-- =============================================================================
CREATE TABLE profile_change_request (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    changes JSONB NOT NULL,
    old_values JSONB,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    machine_result VARCHAR(32),
    machine_reason TEXT,
    reviewer_id VARCHAR(128) REFERENCES user_account(id),
    review_comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TIMESTAMPTZ
);
CREATE INDEX idx_pcr_user_id ON profile_change_request(user_id);
CREATE INDEX idx_pcr_status ON profile_change_request(status);
CREATE INDEX idx_pcr_created ON profile_change_request(created_at DESC);
CREATE INDEX idx_pcr_changes ON profile_change_request USING GIN (changes);

-- =============================================================================
-- Seed data: platform roles, permissions, role-permission bindings
-- =============================================================================
INSERT INTO role (code, name, description, is_system) VALUES
    ('SUPER_ADMIN', 'Super Administrator', 'Full platform access', TRUE),
    ('SKILL_ADMIN', 'Skill Administrator', 'Skill review and management', TRUE),
    ('USER_ADMIN', 'User Administrator', 'User management', TRUE),
    ('AUDITOR', 'Auditor', 'Audit log access', TRUE);

INSERT INTO permission (code, name, group_code) VALUES
    ('skill:publish', 'Publish Skills', 'skill'),
    ('skill:manage', 'Manage Skills', 'skill'),
    ('skill:promote', 'Promote Skills', 'skill'),
    ('review:approve', 'Approve Reviews', 'review'),
    ('promotion:approve', 'Approve Promotions', 'promotion'),
    ('user:manage', 'Manage Users', 'user'),
    ('user:approve', 'Approve Users', 'user'),
    ('audit:read', 'Read Audit Logs', 'audit');

-- SKILL_ADMIN: review:approve, skill:manage, promotion:approve
INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id FROM role r, permission p
WHERE r.code = 'SKILL_ADMIN' AND p.code IN ('review:approve', 'skill:manage', 'promotion:approve');

-- USER_ADMIN: user:manage, user:approve
INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id FROM role r, permission p
WHERE r.code = 'USER_ADMIN' AND p.code IN ('user:manage', 'user:approve');

-- AUDITOR: audit:read
INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id FROM role r, permission p
WHERE r.code = 'AUDITOR' AND p.code = 'audit:read';
