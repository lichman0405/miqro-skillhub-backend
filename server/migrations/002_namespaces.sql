-- Migration 002: Namespaces and namespace membership.
-- Collapses source namespace DDL from V1.
-- All timestamps use TIMESTAMPTZ from the start.

-- =============================================================================
-- Namespaces (GLOBAL or TEAM; lifecycle ACTIVE / FROZEN / ARCHIVED)
-- =============================================================================
CREATE TABLE namespace (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(64) NOT NULL UNIQUE,
    display_name VARCHAR(128) NOT NULL,
    type VARCHAR(32) NOT NULL,
    description TEXT,
    avatar_url VARCHAR(512),
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    created_by VARCHAR(128) REFERENCES user_account(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- =============================================================================
-- Namespace members (OWNER / ADMIN / MEMBER)
-- =============================================================================
CREATE TABLE namespace_member (
    id BIGSERIAL PRIMARY KEY,
    namespace_id BIGINT NOT NULL REFERENCES namespace(id),
    user_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    role VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(namespace_id, user_id)
);
CREATE INDEX idx_namespace_member_user_id ON namespace_member(user_id);
CREATE INDEX idx_namespace_member_namespace_id ON namespace_member(namespace_id);

-- =============================================================================
-- Seed: global namespace
-- =============================================================================
INSERT INTO namespace (slug, display_name, type, description, status)
VALUES ('global', 'Global', 'GLOBAL', 'Platform-level public namespace', 'ACTIVE');
