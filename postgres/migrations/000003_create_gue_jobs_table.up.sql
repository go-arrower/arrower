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
    job_id      TEXT        NOT NULL,           -- no primary key checks improve performance and job_id can be used multiple times, in case of job retry.
    priority    SMALLINT    NOT NULL,
    run_at      TIMESTAMPTZ NOT NULL,
    job_type    TEXT        NOT NULL,
    args        BYTEA       NOT NULL,
    queue       TEXT        NOT NULL,
    run_count   INTEGER     NOT NULL DEFAULT 0, -- how often the job was retried
    run_error   TEXT,                           -- if the job failed, this is it's error
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    success     BOOLEAN     NOT NULL DEFAULT FALSE,
    finished_at TIMESTAMPTZ          DEFAULT NULL
);

SELECT enable_automatic_updated_at('public.gue_jobs_history');


--
CREATE UNLOGGED TABLE public.gue_jobs_worker_pool
(
    id         TEXT        NOT NULL,
    queue      TEXT        NOT NULL,
    workers    SMALLINT    NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE (id, queue)
);


SELECT enable_automatic_updated_at('public.gue_jobs_worker_pool');


-- TODO add SELECT current_database() to run the CRON on there
--CREATE EXTENSION IF NOT EXISTS pg_cron;
--SELECT cron.schedule('arrower:jobs:nightly-vacuum', '0 1 * * *', 'VACUUM public.gue_jobs', 'SELECT current_database()');
--SELECT cron.schedule('arrower:jobs:nightly-vacuum', '0 1 * * *', 'VACUUM public.gue_jobs');
--SELECT cron.schedule('arrower:jobs:nightly-worker-clean', '0 2 * * *', $$DELETE FROM public.gue_jobs_worker_pool WHERE updated_at < now() - interval '1 week'$$);


COMMIT;