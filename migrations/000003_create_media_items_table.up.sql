-- Migration: 000003_create_media_items_table.up.sql
CREATE TYPE media_status AS ENUM ('PENDING', 'DOWNLOADING', 'PROCESSING', 'COMPLETED', 'FAILED');

CREATE TABLE media_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    title           VARCHAR(500),
    source_url      TEXT NOT NULL,
    status          media_status NOT NULL DEFAULT 'PENDING',
    progress        INT NOT NULL DEFAULT 0,
    size_bytes      BIGINT,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_media_items_job_id ON media_items(job_id);
CREATE INDEX idx_media_items_status ON media_items(status);

-- name: down
DROP TABLE IF EXISTS media_items;
DROP TYPE IF EXISTS media_status;