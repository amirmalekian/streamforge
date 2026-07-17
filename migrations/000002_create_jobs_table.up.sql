-- Migration: 000002_create_jobs_table.up.sql
CREATE TYPE job_status AS ENUM ('CREATED', 'QUEUED', 'PROCESSING', 'COMPLETED', 'FAILED', 'CANCELLED');

CREATE TABLE jobs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_url          TEXT NOT NULL,
    status              job_status NOT NULL DEFAULT 'CREATED',
    total_items         INT NOT NULL DEFAULT 0,
    completed_items     INT NOT NULL DEFAULT 0,
    error_message       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_jobs_user_id ON jobs(user_id);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);

-- name: down
DROP TABLE IF EXISTS jobs;
DROP TYPE IF EXISTS job_status;