CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE import_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id       UUID NOT NULL,
    file_key        TEXT NOT NULL,
    status          VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    total_rows      INT,
    rows_processed  INT NOT NULL DEFAULT 0,
    rows_failed     INT NOT NULL DEFAULT 0,
    error_file_key  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_import_jobs_seller ON import_jobs(seller_id);
CREATE INDEX idx_import_jobs_status ON import_jobs(status);
