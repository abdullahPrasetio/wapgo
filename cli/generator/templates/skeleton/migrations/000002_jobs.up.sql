-- 000002_jobs: Postgres-backed job queue used by cmd/worker.
-- Jobs are claimed with SELECT ... FOR UPDATE SKIP LOCKED.

CREATE TABLE jobs (
    id           BIGSERIAL    PRIMARY KEY,
    kind         VARCHAR(100) NOT NULL,
    payload      JSONB        NOT NULL DEFAULT '{}'::jsonb,
    status       VARCHAR(16)  NOT NULL DEFAULT 'pending',
    attempts     INT          NOT NULL DEFAULT 0,
    max_attempts INT          NOT NULL DEFAULT 5,
    run_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    locked_at    TIMESTAMPTZ  NULL,
    locked_by    VARCHAR(200) NULL,
    last_error   TEXT         NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT jobs_status_check CHECK (status IN ('pending', 'running', 'done', 'failed'))
);

CREATE INDEX idx_jobs_claim ON jobs (status, run_at);
CREATE INDEX idx_jobs_running_locked_at ON jobs (locked_at) WHERE status = 'running';
