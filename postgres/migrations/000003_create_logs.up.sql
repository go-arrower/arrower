BEGIN;


CREATE UNLOGGED TABLE IF NOT EXISTS public.log
(
    time    TIMESTAMP WITH TIME ZONE NOT NULL,
    user_id UUID DEFAULT NULL,
    log     JSONB                    NOT NULL
);

CREATE INDEX IF NOT EXISTS log_time_idx ON public.log(time);

COMMIT;