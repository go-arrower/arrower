BEGIN;


-- use gue as job worker in postgres, see: https://github.com/vgarvardt/gue
-- sql migration taken from: https://github.com/vgarvardt/gue/blob/master/migrations/schema.sql
CREATE TABLE IF NOT EXISTS public.gue_jobs
(
    job_id      TEXT        NOT NULL PRIMARY KEY,
    priority    SMALLINT    NOT NULL,
    run_at      TIMESTAMPTZ NOT NULL,
    job_type    TEXT        NOT NULL,
    args        BYTEA       NOT NULL,
    error_count INTEGER     NOT NULL DEFAULT 0,
    last_error  TEXT,
    queue       TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_gue_jobs_selector ON gue_jobs (queue, run_at, priority);

SELECT enable_automatic_updated_at('public.gue_jobs');


-- collect historic gue_jobs for analytics, as the workers remove them from gue_jobs table after success.
CREATE TABLE IF NOT EXISTS public.gue_jobs_history
(
    job_id      TEXT        NOT NULL, -- no primary key checks improve performance and job_id can be used multiple times, in case of job retry.
    priority    SMALLINT    NOT NULL,
    run_at      TIMESTAMPTZ NOT NULL,
    job_type    TEXT        NOT NULL,
    args        BYTEA       NOT NULL,
    error_count INTEGER     NOT NULL DEFAULT 0,
    last_error  TEXT,
    queue       TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,

    success     BOOLEAN     NOT NULL DEFAULT FALSE,
    finished_at TIMESTAMPTZ          DEFAULT NULL
);

SELECT enable_automatic_updated_at('public.gue_jobs_history');


COMMIT;