-- Migration 004: Review, promotion, governance notifications, idempotency.
-- Collapses source V3, V6 (partial), V12, V37.
-- All timestamps use TIMESTAMPTZ from the start.

-- =============================================================================
-- Review tasks
-- =============================================================================
CREATE TABLE review_task (
    id BIGSERIAL PRIMARY KEY,
    skill_version_id BIGINT NOT NULL REFERENCES skill_version(id),
    namespace_id BIGINT NOT NULL REFERENCES namespace(id),
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    version INT NOT NULL DEFAULT 1,
    submitted_by VARCHAR(128) NOT NULL REFERENCES user_account(id),
    reviewed_by VARCHAR(128) REFERENCES user_account(id),
    review_comment TEXT,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TIMESTAMPTZ
);
CREATE INDEX idx_review_task_namespace_status ON review_task(namespace_id, status);
CREATE INDEX idx_review_task_submitted_by_status ON review_task(submitted_by, status);
CREATE UNIQUE INDEX idx_review_task_version_pending
    ON review_task(skill_version_id) WHERE status = 'PENDING';

-- =============================================================================
-- Promotion requests
-- =============================================================================
CREATE TABLE promotion_request (
    id BIGSERIAL PRIMARY KEY,
    source_skill_id BIGINT NOT NULL REFERENCES skill(id),
    source_version_id BIGINT NOT NULL REFERENCES skill_version(id),
    target_namespace_id BIGINT NOT NULL REFERENCES namespace(id),
    target_skill_id BIGINT REFERENCES skill(id),
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    version INT NOT NULL DEFAULT 1,
    submitted_by VARCHAR(128) NOT NULL REFERENCES user_account(id),
    reviewed_by VARCHAR(128) REFERENCES user_account(id),
    review_comment TEXT,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TIMESTAMPTZ
);
CREATE INDEX idx_promotion_request_source_skill ON promotion_request(source_skill_id);
CREATE INDEX idx_promotion_request_status ON promotion_request(status);
CREATE INDEX idx_promotion_request_target_namespace ON promotion_request(target_namespace_id);
CREATE UNIQUE INDEX idx_promotion_request_version_pending
    ON promotion_request(source_version_id) WHERE status = 'PENDING';

-- =============================================================================
-- User notifications (governance workbench)
-- =============================================================================
CREATE TABLE user_notification (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(128) NOT NULL,
    category VARCHAR(64) NOT NULL,
    entity_type VARCHAR(64) NOT NULL,
    entity_id BIGINT NOT NULL,
    title VARCHAR(200) NOT NULL,
    body_json TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'UNREAD',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read_at TIMESTAMPTZ
);
CREATE INDEX idx_user_notification_user_created_at ON user_notification(user_id, created_at DESC);
CREATE INDEX idx_user_notification_user_status ON user_notification(user_id, status, created_at DESC);

-- =============================================================================
-- System notifications
-- =============================================================================
CREATE TABLE notification (
    id BIGSERIAL PRIMARY KEY,
    recipient_id VARCHAR(128) NOT NULL,
    category VARCHAR(32) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    title VARCHAR(200) NOT NULL,
    body_json TEXT,
    entity_type VARCHAR(64),
    entity_id BIGINT,
    status VARCHAR(20) NOT NULL DEFAULT 'UNREAD',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read_at TIMESTAMPTZ
);
CREATE INDEX idx_notification_recipient_created ON notification(recipient_id, created_at DESC);
CREATE INDEX idx_notification_recipient_status ON notification(recipient_id, status, created_at DESC);

-- =============================================================================
-- Notification preferences
-- =============================================================================
CREATE TABLE notification_preference (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(128) NOT NULL,
    category VARCHAR(32) NOT NULL,
    channel VARCHAR(32) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE(user_id, category, channel)
);

-- =============================================================================
-- Idempotency records
-- =============================================================================
CREATE TABLE idempotency_record (
    request_id VARCHAR(64) PRIMARY KEY,
    resource_type VARCHAR(64) NOT NULL,
    resource_id BIGINT,
    status VARCHAR(32) NOT NULL,
    response_status_code INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_idempotency_record_expires_at ON idempotency_record(expires_at);
CREATE INDEX idx_idempotency_record_status_created ON idempotency_record(status, created_at);
