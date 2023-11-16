BEGIN;


DROP EXTENSION IF EXISTS pg_cron;




DROP FUNCTION IF EXISTS cron.unschedule(job_name TEXT);
DROP FUNCTION IF EXISTS cron.unschedule(job_id BIGINT);
DROP FUNCTION IF EXISTS cron.schedule_in_database(job_name TEXT, schedule TEXT, command TEXT, database TEXT, username TEXT, active BOOLEAN);
DROP FUNCTION IF EXISTS cron.schedule(schedule TEXT, command TEXT);
DROP FUNCTION IF EXISTS cron.schedule(job_name TEXT, schedule TEXT, command TEXT);
DROP FUNCTION IF EXISTS cron.alter_job(job_id BIGINT, schedule TEXT, command TEXT, database TEXT, username TEXT, active BOOLEAN);

DROP SERVER IF EXISTS localhost_postgres_db CASCADE;

DROP EXTENSION IF EXISTS dblink;
DROP EXTENSION IF EXISTS postgres_fdw;

DROP SCHEMA IF EXISTS cron CASCADE;


COMMIT;