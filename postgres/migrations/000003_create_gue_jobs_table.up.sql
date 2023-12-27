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
    last_error  TEXT        NOT NULL DEFAULT '',
    queue       TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_gue_jobs_selector ON gue_jobs (queue, run_at, priority);

SELECT enable_automatic_updated_at('public.gue_jobs');

SELECT cron.schedule('arrower:jobs:nightly-vacuum', '0 1 * * *', 'VACUUM public.gue_jobs');


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

SELECT cron.schedule('arrower:jobs:nightly-worker-clean', '0 2 * * *',
                     $$DELETE FROM public.gue_jobs_worker_pool WHERE updated_at < now() - interval '1 week'$$);


CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- reimplement the ulid generation of the underlying Go library, to manually create valid job ids.
CREATE OR REPLACE FUNCTION generate_ulid() RETURNS TEXT AS
$$
DECLARE
    -- Crockford's Base32
    encoding  BYTEA = '0123456789ABCDEFGHJKMNPQRSTVWXYZ';
    timestamp BYTEA = E'\\000\\000\\000\\000\\000\\000';
    output    TEXT  = '';
    unix_time BIGINT;
    ulid      BYTEA;
BEGIN
    unix_time = (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT;
    timestamp = SET_BYTE(timestamp, 0, (unix_time >> 40)::BIT(8)::INTEGER);
    timestamp = SET_BYTE(timestamp, 1, (unix_time >> 32)::BIT(8)::INTEGER);
    timestamp = SET_BYTE(timestamp, 2, (unix_time >> 24)::BIT(8)::INTEGER);
    timestamp = SET_BYTE(timestamp, 3, (unix_time >> 16)::BIT(8)::INTEGER);
    timestamp = SET_BYTE(timestamp, 4, (unix_time >> 8)::BIT(8)::INTEGER);
    timestamp = SET_BYTE(timestamp, 5, unix_time::BIT(8)::INTEGER);

    ulid = timestamp || gen_random_bytes(10);

    -- 10 byte timestamp
    output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 0) & 224) >> 5));
    output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 0) & 31)));
    output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 1) & 248) >> 3));
    output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 1) & 7) << 2) | ((GET_BYTE(ulid, 2) & 192) >> 6)));
    output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 2) & 62) >> 1));
    output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 2) & 1) << 4) | ((GET_BYTE(ulid, 3) & 240) >> 4)));
    output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 3) & 15) << 1) | ((GET_BYTE(ulid, 4) & 128) >> 7)));
    output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 4) & 124) >> 2));
    output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 4) & 3) << 3) | ((GET_BYTE(ulid, 5) & 224) >> 5)));
    output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 5) & 31)));

    -- 16 bytes of entropy
    output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 6) & 248) >> 3));
    output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 6) & 7) << 2) | ((GET_BYTE(ulid, 7) & 192) >> 6)));
    output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 7) & 62) >> 1));
    output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 7) & 1) << 4) | ((GET_BYTE(ulid, 8) & 240) >> 4)));
    output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 8) & 15) << 1) | ((GET_BYTE(ulid, 9) & 128) >> 7)));
    output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 9) & 124) >> 2));
    output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 9) & 3) << 3) | ((GET_BYTE(ulid, 10) & 224) >> 5)));
    output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 10) & 31)));
    output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 11) & 248) >> 3));
    output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 11) & 7) << 2) | ((GET_BYTE(ulid, 12) & 192) >> 6)));
    output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 12) & 62) >> 1));
    output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 12) & 1) << 4) | ((GET_BYTE(ulid, 13) & 240) >> 4)));
    output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 13) & 15) << 1) | ((GET_BYTE(ulid, 14) & 128) >> 7)));
    output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 14) & 124) >> 2));
    output = output || CHR(GET_BYTE(encoding, ((GET_BYTE(ulid, 14) & 3) << 3) | ((GET_BYTE(ulid, 15) & 224) >> 5)));
    output = output || CHR(GET_BYTE(encoding, (GET_BYTE(ulid, 15) & 31)));

    RETURN output;
END;
$$ LANGUAGE plpgsql VOLATILE;


COMMIT;