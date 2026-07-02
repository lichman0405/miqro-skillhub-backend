-- Migration 007: Community features — skill issues, discussions, wiki pages,
-- change proposals, community labels, and moderation.
-- All timestamps use TIMESTAMPTZ.

-- =============================================================================
-- Skill issues
-- =============================================================================
CREATE TABLE skill_issue (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL REFERENCES skill(id) ON DELETE CASCADE,
    title VARCHAR(512) NOT NULL,
    body TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'CLOSED')),
    assignee_id VARCHAR(128) REFERENCES user_account(id),
    linked_version_id BIGINT REFERENCES skill_version(id),
    linked_release_id BIGINT REFERENCES skill_release(id),
    author_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    locked BOOLEAN NOT NULL DEFAULT FALSE,
    comment_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_skill_issue_skill_id ON skill_issue(skill_id, status, created_at DESC);
CREATE INDEX idx_skill_issue_author ON skill_issue(author_id, created_at DESC);
CREATE INDEX idx_skill_issue_assignee ON skill_issue(assignee_id, status);
CREATE INDEX idx_skill_issue_linked_version ON skill_issue(linked_version_id);
CREATE INDEX idx_skill_issue_linked_release ON skill_issue(linked_release_id);

-- =============================================================================
-- Skill issue comments
-- =============================================================================
CREATE TABLE skill_issue_comment (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT NOT NULL REFERENCES skill_issue(id) ON DELETE CASCADE,
    author_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_issue_comment_issue_id ON skill_issue_comment(issue_id, created_at);

-- =============================================================================
-- Skill discussions
-- =============================================================================
CREATE TABLE skill_discussion (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL REFERENCES skill(id) ON DELETE CASCADE,
    title VARCHAR(512) NOT NULL,
    body TEXT,
    category VARCHAR(32) NOT NULL DEFAULT 'GENERAL'
        CHECK (category IN ('GENERAL', 'QA', 'IDEAS', 'ANNOUNCEMENTS')),
    accepted_answer_id BIGINT,
    author_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    locked BOOLEAN NOT NULL DEFAULT FALSE,
    pinned BOOLEAN NOT NULL DEFAULT FALSE,
    comment_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_skill_discussion_skill_id ON skill_discussion(skill_id, category, pinned DESC, created_at DESC);
CREATE INDEX idx_skill_discussion_author ON skill_discussion(author_id, created_at DESC);

-- =============================================================================
-- Skill discussion comments
-- =============================================================================
CREATE TABLE skill_discussion_comment (
    id BIGSERIAL PRIMARY KEY,
    discussion_id BIGINT NOT NULL REFERENCES skill_discussion(id) ON DELETE CASCADE,
    author_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    body TEXT NOT NULL,
    is_answer BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_discussion_comment_discussion_id ON skill_discussion_comment(discussion_id, created_at);

ALTER TABLE skill_discussion ADD CONSTRAINT fk_discussion_accepted_answer
    FOREIGN KEY (accepted_answer_id) REFERENCES skill_discussion_comment(id)
    DEFERRABLE INITIALLY DEFERRED;

-- =============================================================================
-- Skill wiki pages
-- =============================================================================
CREATE TABLE skill_wiki_page (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL REFERENCES skill(id) ON DELETE CASCADE,
    title VARCHAR(512) NOT NULL,
    slug VARCHAR(128) NOT NULL,
    current_version_id BIGINT,
    order_index INT NOT NULL DEFAULT 0,
    author_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(skill_id, slug)
);
CREATE INDEX idx_wiki_page_skill_id ON skill_wiki_page(skill_id, order_index);

-- =============================================================================
-- Skill wiki page versions (history)
-- =============================================================================
CREATE TABLE skill_wiki_page_version (
    id BIGSERIAL PRIMARY KEY,
    page_id BIGINT NOT NULL REFERENCES skill_wiki_page(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    version INT NOT NULL DEFAULT 1,
    change_summary VARCHAR(256),
    linked_skill_version_id BIGINT REFERENCES skill_version(id),
    author_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_wiki_page_version_page_id ON skill_wiki_page_version(page_id, version DESC);

ALTER TABLE skill_wiki_page ADD CONSTRAINT fk_wiki_page_current_version
    FOREIGN KEY (current_version_id) REFERENCES skill_wiki_page_version(id)
    DEFERRABLE INITIALLY DEFERRED;

-- =============================================================================
-- Skill change proposals (skill-native collaboration)
-- =============================================================================
CREATE TABLE skill_change_proposal (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL REFERENCES skill(id) ON DELETE CASCADE,
    title VARCHAR(512) NOT NULL,
    summary TEXT,
    proposed_changes_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL DEFAULT 'OPEN'
        CHECK (status IN ('OPEN', 'ACCEPTED', 'REJECTED', 'WITHDRAWN')),
    author_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    reviewer_id VARCHAR(128) REFERENCES user_account(id),
    source_git_ref VARCHAR(512),
    review_comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_change_proposal_skill_id ON skill_change_proposal(skill_id, status, created_at DESC);
CREATE INDEX idx_change_proposal_author ON skill_change_proposal(author_id, created_at DESC);

-- =============================================================================
-- Skill change proposal comments
-- =============================================================================
CREATE TABLE skill_change_proposal_comment (
    id BIGSERIAL PRIMARY KEY,
    proposal_id BIGINT NOT NULL REFERENCES skill_change_proposal(id) ON DELETE CASCADE,
    author_id VARCHAR(128) NOT NULL REFERENCES user_account(id),
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_proposal_comment_proposal_id ON skill_change_proposal_comment(proposal_id, created_at);

-- =============================================================================
-- Community labels — reuse label_definition for issues and discussions
-- =============================================================================
CREATE TABLE issue_label (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT NOT NULL REFERENCES skill_issue(id) ON DELETE CASCADE,
    label_id BIGINT NOT NULL REFERENCES label_definition(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(issue_id, label_id)
);
CREATE INDEX idx_issue_label_label_id ON issue_label(label_id);

CREATE TABLE discussion_label (
    id BIGSERIAL PRIMARY KEY,
    discussion_id BIGINT NOT NULL REFERENCES skill_discussion(id) ON DELETE CASCADE,
    label_id BIGINT NOT NULL REFERENCES label_definition(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(discussion_id, label_id)
);
CREATE INDEX idx_discussion_label_label_id ON discussion_label(label_id);

-- =============================================================================
-- Community moderation reports — extend skill_report to community objects
-- =============================================================================
CREATE TABLE community_report (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL REFERENCES skill(id) ON DELETE CASCADE,
    object_type VARCHAR(32) NOT NULL CHECK (object_type IN ('ISSUE', 'DISCUSSION', 'COMMENT', 'WIKI_PAGE')),
    object_id BIGINT NOT NULL,
    reporter_id VARCHAR(128) NOT NULL,
    reason VARCHAR(200) NOT NULL,
    details TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    handled_by VARCHAR(128),
    handle_comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    handled_at TIMESTAMPTZ
);
CREATE INDEX idx_community_report_object ON community_report(object_type, object_id);
CREATE INDEX idx_community_report_status ON community_report(status, created_at DESC);
