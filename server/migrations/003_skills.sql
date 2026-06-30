-- Migration 003: Skills, versions, files, tags, stats, storage compensation, search documents.
-- Collapses source V2, V4, V6, V9, V11, V13, V14, V15, V28, V29, V30, V31, V32, V33, V40.
-- All timestamps use TIMESTAMPTZ from the start.

-- =============================================================================
-- Skills
-- =============================================================================
CREATE TABLE skill (
    id BIGSERIAL PRIMARY KEY,
    namespace_id BIGINT NOT NULL REFERENCES namespace(id),
    slug VARCHAR(128) NOT NULL,
    display_name VARCHAR(256),
    summary TEXT,
    owner_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    source_skill_id BIGINT,
    visibility VARCHAR(32) NOT NULL DEFAULT 'PUBLIC',
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    latest_version_id BIGINT,
    download_count BIGINT NOT NULL DEFAULT 0,
    star_count INT NOT NULL DEFAULT 0,
    rating_avg DECIMAL(3,2) NOT NULL DEFAULT 0.00,
    rating_count INT NOT NULL DEFAULT 0,
    subscription_count INT NOT NULL DEFAULT 0,
    hidden BOOLEAN NOT NULL DEFAULT FALSE,
    hidden_at TIMESTAMPTZ,
    hidden_by VARCHAR(128) REFERENCES user_account(id),
    created_by VARCHAR(128) REFERENCES user_account(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(128) REFERENCES user_account(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(namespace_id, slug, owner_id)
);
CREATE INDEX idx_skill_namespace_status ON skill(namespace_id, status);
CREATE INDEX idx_skill_hidden ON skill(hidden) WHERE hidden = TRUE;
CREATE INDEX idx_skill_active_visible_updated
    ON skill(updated_at DESC, id DESC) WHERE status = 'ACTIVE' AND hidden = FALSE;
CREATE INDEX idx_skill_active_visible_downloads
    ON skill(download_count DESC, updated_at DESC, id DESC) WHERE status = 'ACTIVE' AND hidden = FALSE;
CREATE INDEX idx_skill_active_visible_rating
    ON skill(rating_avg DESC, updated_at DESC, id DESC) WHERE status = 'ACTIVE' AND hidden = FALSE;

-- =============================================================================
-- Skill versions
-- =============================================================================
CREATE TABLE skill_version (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL REFERENCES skill(id),
    version VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'DRAFT',
    changelog TEXT,
    parsed_metadata_json JSONB,
    manifest_json JSONB,
    requested_visibility VARCHAR(32),
    file_count INT NOT NULL DEFAULT 0,
    total_size BIGINT NOT NULL DEFAULT 0,
    bundle_ready BOOLEAN NOT NULL DEFAULT FALSE,
    download_ready BOOLEAN NOT NULL DEFAULT FALSE,
    published_at TIMESTAMPTZ,
    yanked_at TIMESTAMPTZ,
    yanked_by VARCHAR(128) REFERENCES user_account(id),
    yank_reason TEXT,
    created_by VARCHAR(128) NOT NULL REFERENCES user_account(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(skill_id, version)
);
CREATE INDEX idx_skill_version_skill_status ON skill_version(skill_id, status);

-- FK from skill.latest_version_id -> skill_version.id (deferred to avoid circular dependency)
ALTER TABLE skill ADD CONSTRAINT fk_skill_latest_version
    FOREIGN KEY (latest_version_id) REFERENCES skill_version(id);

-- =============================================================================
-- Skill files
-- =============================================================================
CREATE TABLE skill_file (
    id BIGSERIAL PRIMARY KEY,
    version_id BIGINT NOT NULL REFERENCES skill_version(id),
    file_path VARCHAR(512) NOT NULL,
    file_size BIGINT NOT NULL,
    content_type VARCHAR(128),
    sha256 VARCHAR(64) NOT NULL,
    storage_key VARCHAR(512) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(version_id, file_path)
);

-- =============================================================================
-- Skill tags
-- =============================================================================
CREATE TABLE skill_tag (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL REFERENCES skill(id),
    tag_name VARCHAR(64) NOT NULL,
    version_id BIGINT NOT NULL REFERENCES skill_version(id),
    created_by VARCHAR(128) REFERENCES user_account(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(skill_id, tag_name)
);

-- =============================================================================
-- Skill version stats (per-version download counters)
-- =============================================================================
CREATE TABLE skill_version_stats (
    skill_version_id BIGINT PRIMARY KEY REFERENCES skill_version(id) ON DELETE CASCADE,
    skill_id BIGINT NOT NULL REFERENCES skill(id) ON DELETE CASCADE,
    download_count BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_skill_version_stats_skill_id ON skill_version_stats(skill_id);

-- =============================================================================
-- Storage deletion compensation
-- =============================================================================
CREATE TABLE skill_storage_delete_compensation (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT,
    namespace VARCHAR(128) NOT NULL,
    slug VARCHAR(128) NOT NULL,
    storage_keys_json TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    attempt_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    last_attempt_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_skill_storage_delete_comp_status_created
    ON skill_storage_delete_compensation(status, created_at);

-- =============================================================================
-- Skill search document (denormalized, with generated tsvector)
-- =============================================================================
CREATE TABLE skill_search_document (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL UNIQUE REFERENCES skill(id),
    namespace_id BIGINT NOT NULL,
    namespace_slug VARCHAR(64) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    title VARCHAR(512),
    summary TEXT,
    keywords TEXT,
    search_text TEXT,
    semantic_vector TEXT,
    visibility VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('simple', coalesce(summary, '')), 'B') ||
        setweight(to_tsvector('simple', coalesce(keywords, '')), 'B') ||
        setweight(to_tsvector('simple', coalesce(search_text, '')), 'C')
    ) STORED
);
CREATE INDEX idx_search_vector ON skill_search_document USING GIN (search_vector);
CREATE INDEX idx_search_doc_namespace ON skill_search_document(namespace_id);
CREATE INDEX idx_search_doc_visibility ON skill_search_document(visibility);
