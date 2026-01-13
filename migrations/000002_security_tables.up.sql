-- Audit Logs Table (Immutable for SOX compliance)
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id UUID NOT NULL,
    user_role VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255) NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    details JSONB,
    success BOOLEAN NOT NULL,
    error_message TEXT
);
-- Index for querying by user and time
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_time ON audit_logs (user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs (action, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs (resource_type, resource_id);
-- Processed Messages Table (Idempotency for Kafka)
CREATE TABLE IF NOT EXISTS processed_messages (
    id UUID PRIMARY KEY,
    topic VARCHAR(255) NOT NULL,
    partition INTEGER NOT NULL,
    offset_id BIGINT NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(topic, partition, offset_id)
);
-- Index for fast lookups
CREATE INDEX IF NOT EXISTS idx_processed_messages_lookup ON processed_messages (topic, partition, offset_id);
-- Retention: Auto-delete old processed messages after 7 days (configurable)
-- In production, consider using TimescaleDB's retention policies