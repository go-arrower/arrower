BEGIN;


CREATE UNLOGGED TABLE IF NOT EXISTS arrower.log
(
    time    TIMESTAMP WITH TIME ZONE NOT NULL,
    user_id UUID DEFAULT NULL,
    log     JSONB                    NOT NULL
);

CREATE INDEX IF NOT EXISTS log_time_idx ON arrower.log(time);

SELECT enable_automatic_updated_at('arrower.log');

COMMIT;