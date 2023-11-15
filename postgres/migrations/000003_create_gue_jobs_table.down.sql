BEGIN;


SELECT cron.unschedule('arrower:jobs:nightly-worker-clean' );
SELECT cron.unschedule('arrower:jobs:nightly-vacuum' );


DROP TABLE IF EXISTS public.gue_jobs;
DROP TABLE IF EXISTS public.gue_jobs_history;
DROP TABLE IF EXISTS public.gue_jobs_worker_pool;


COMMIT;