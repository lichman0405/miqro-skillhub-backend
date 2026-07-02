-- Migration 006: Skill releases.
-- Adds first-class release objects on top of published skill versions.
-- Each published version can have one stable release per channel.

-- =============================================================================
-- Skill release
-- =============================================================================
CREATE TABLE skill_release (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL REFERENCES skill(id),
    version_id BIGINT NOT NULL REFERENCES skill_version(id),
    channel VARCHAR(64) NOT NULL DEFAULT 'stable',
    title VARCHAR(512) NOT NULL,
    notes TEXT,
    draft BOOLEAN NOT NULL DEFAULT FALSE,
    prerelease BOOLEAN NOT NULL DEFAULT FALSE,
    yanked BOOLEAN NOT NULL DEFAULT FALSE,
    published_at TIMESTAMPTZ,
    publisher_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    reviewer_id VARCHAR(128) REFERENCES user_account(id),
    package_hash VARCHAR(128),
    ci_check_run_id VARCHAR(256),
    metadata_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- One stable (non-draft) release per version per channel.
    UNIQUE(version_id, channel)
);
CREATE INDEX idx_skill_release_skill ON skill_release(skill_id);
CREATE INDEX idx_skill_release_channel_published ON skill_release(channel, published_at DESC)
    WHERE draft = FALSE AND yanked = FALSE;
CREATE INDEX idx_skill_release_version ON skill_release(version_id);

-- =============================================================================
-- Release assets
-- =============================================================================
CREATE TABLE skill_release_asset (
    id BIGSERIAL PRIMARY KEY,
    release_id BIGINT NOT NULL REFERENCES skill_release(id) ON DELETE CASCADE,
    name VARCHAR(512) NOT NULL,
    label VARCHAR(256),
    content_type VARCHAR(128) NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    storage_key VARCHAR(512) NOT NULL,
    sha256 VARCHAR(64),
    download_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_skill_release_asset_release ON skill_release_asset(release_id);
