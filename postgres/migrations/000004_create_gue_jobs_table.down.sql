BEGIN;


DROP FUNCTION IF EXISTS generate_ulid;
DROP EXTENSION IF EXISTS pgcrypto;


SELECT cron.unschedule('arrower:jobs:nightly-worker-clean' );
SELECT cron.unschedule('arrower:jobs:nightly-vacuum' );


DROP TABLE IF EXISTS arrower.gue_jobs;
DROP TABLE IF EXISTS arrower.gue_jobs_history;
DROP TABLE IF EXISTS arrower.gue_jobs_worker_pool;


COMMIT;