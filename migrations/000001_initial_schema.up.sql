-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- Reports Table
CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    format VARCHAR(10) NOT NULL,
    generated_by UUID NOT NULL,
    filters JSONB,
    storage_path TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    error TEXT
);
-- Transaction Metrics (Fact Table)
CREATE TABLE IF NOT EXISTS transaction_metrics (
    time TIMESTAMPTZ NOT NULL,
    transaction_id UUID NOT NULL,
    amount DECIMAL(20, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    latency_ms INTEGER,
    source_account UUID,
    target_account UUID
);
-- Convert to hypertable
SELECT create_hypertable(
        'transaction_metrics',
        'time',
        if_not_exists => TRUE
    );
-- Indexes for common analytics
CREATE INDEX IF NOT EXISTS idx_transaction_metrics_cust_time ON transaction_metrics (source_account, time DESC);
CREATE INDEX IF NOT EXISTS idx_transaction_metrics_type_time ON transaction_metrics (type, time DESC);