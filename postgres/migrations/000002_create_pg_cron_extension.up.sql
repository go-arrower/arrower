BEGIN;


--
-- Setup pg_cron extension. If this is not possible as pg_cron can only be installed and used in one database, this
-- sets up a workaround simulating pg_cron in the current db.
--

DO $$
BEGIN
    CREATE EXTENSION IF NOT EXISTS pg_cron;

    EXCEPTION -- (details: ERROR: can only create extension in database postgres (SQLSTATE P0001))
        -- It is assumed that pg_cron is installed in the default database 'postgres' and this user has access to it.
        -- The following will simulate pg_cron as if it was installed in this database. The main purpose is to run automated arrower integration tests.
        WHEN SQLSTATE 'P0001' THEN
            CREATE SCHEMA IF NOT EXISTS cron;

            CREATE EXTENSION IF NOT EXISTS postgres_fdw;
            CREATE EXTENSION IF NOT EXISTS dblink;


            CREATE SERVER IF NOT EXISTS localhost_postgres_db FOREIGN DATA WRAPPER postgres_fdw OPTIONS (dbname 'postgres');
            DO $fdw$
                BEGIN
                    EXECUTE FORMAT($m$CREATE USER MAPPING IF NOT EXISTS FOR CURRENT_USER SERVER localhost_postgres_db OPTIONS (user '%s');$m$, CURRENT_USER);
                END
            $fdw$ LANGUAGE plpgsql;
            -- EXECUTE FORMAT has to be used:
            -- Variable substitution currently works only in SELECT, INSERT, UPDATE, and DELETE commands, because the main SQL
            -- engine allows query parameters only in these commands. To use a non-constant name or value in other statement types
            -- (generically called utility statements), you must construct the utility statement as a string and EXECUTE it.
            -- See: https://www.postgresql.org/docs/current/plpgsql-implementation.html#PLPGSQL-VAR-SUBST


            -- Importing the whole schema also shows jobs that belong to a different user or db.
            -- Although CREATE FOREIGN TABLE could be used to filter out those rows, this behaviour is closer to
            -- the original pg_cron behaviour.
            -- consequence: pgAdmin will show the foreign table as read only, but it can be inserted into and updated via SQL
            IMPORT FOREIGN SCHEMA cron FROM SERVER localhost_postgres_db INTO cron;


            -- Create all pg_cron functions by proxying them to the original
            CREATE OR REPLACE FUNCTION cron.alter_job(job_id BIGINT, schedule TEXT, command TEXT, database TEXT, username TEXT, active BOOLEAN) RETURNS VOID AS $fn$
            DECLARE
                sql        TEXT;
                connstr    NAME;
            BEGIN
                connstr := 'dbname=postgres user=' || CURRENT_USER;

                sql := FORMAT('SELECT cron.alter_job(%L,%L,%L,%L,%L,%L);', job_id, schedule, command, database, username, active);
                EXECUTE 'SELECT * FROM dblink($1,$2,true) AS (schedule BIGINT);'
                    USING connstr,sql;
            END
            $fn$ LANGUAGE PLPGSQL;


            CREATE OR REPLACE FUNCTION cron.schedule(job_name TEXT, schedule TEXT, command TEXT) RETURNS BIGINT AS $fn$
            DECLARE
                sql        TEXT;
                database   NAME;
                connstr    NAME;
                cronresult BIGINT;
            BEGIN
                connstr := 'dbname=postgres user=' || CURRENT_USER;
                SELECT CURRENT_DATABASE() INTO database;

                sql := FORMAT('SELECT cron.schedule_in_database(%L,%L,%L,%L,%L,%L);', job_name, schedule, command, database, CURRENT_USER, true);
                EXECUTE 'SELECT * FROM dblink($1,$2,true) AS (schedule BIGINT);'
                    USING connstr,sql
                    INTO cronresult;

                RETURN cronresult;
            END
            $fn$ LANGUAGE PLPGSQL;


            CREATE OR REPLACE FUNCTION cron.schedule(schedule TEXT, command TEXT) RETURNS BIGINT AS $fn$
            DECLARE
                sql        TEXT;
                database   NAME;
                connstr    NAME;
                cronresult BIGINT;
            BEGIN
                connstr := 'dbname=postgres user=' || CURRENT_USER;
                SELECT CURRENT_DATABASE() INTO database;

                sql := FORMAT('SELECT cron.schedule_in_database(%L,%L,%L,%L,%L,%L);', '', schedule, command, database, CURRENT_USER, true);
                EXECUTE 'SELECT * FROM dblink($1,$2,true) AS (schedule BIGINT);'
                    USING connstr,sql
                    INTO cronresult;

                RETURN cronresult;
            END
            $fn$ LANGUAGE PLPGSQL;


            CREATE OR REPLACE FUNCTION cron.schedule_in_database(job_name TEXT, schedule TEXT, command TEXT, database TEXT, username TEXT, active BOOLEAN) RETURNS BIGINT AS $fn$
            DECLARE
                sql        TEXT;
                connstr    NAME;
                cronresult BIGINT;
            BEGIN
                connstr := 'dbname=postgres user=' || CURRENT_USER;

                sql := FORMAT('SELECT cron.schedule_in_database(%L,%L,%L,%L,%L,%L);', job_name, schedule, command, database, username, active);
                EXECUTE 'SELECT * FROM dblink($1,$2,true) AS (schedule BIGINT);'
                    USING connstr,sql
                    INTO cronresult;

                RETURN cronresult;
            END
            $fn$ LANGUAGE PLPGSQL;


            CREATE OR REPLACE FUNCTION cron.unschedule(job_id BIGINT) RETURNS BOOLEAN AS $fn$
            DECLARE
                sql        TEXT;
                connstr    NAME;
                cronresult BOOLEAN;
            BEGIN
                connstr := 'dbname=postgres user=' || CURRENT_USER;

                sql := FORMAT('SELECT cron.unschedule(%s);', job_id);
                EXECUTE 'SELECT * FROM dblink($1,$2,true) AS (unschedule BOOLEAN);'
                    USING connstr,sql
                    INTO cronresult;

                RETURN cronresult;
            END
            $fn$ LANGUAGE PLPGSQL;


            CREATE OR REPLACE FUNCTION cron.unschedule(job_name TEXT) RETURNS BOOLEAN AS $fn$
            DECLARE
                sql        TEXT;
                connstr    NAME;
                cronresult BOOLEAN;
            BEGIN
                connstr := 'dbname=postgres user=' || CURRENT_USER;

                sql := FORMAT('SELECT cron.unschedule(%L);', job_name);
                EXECUTE 'SELECT * FROM dblink($1,$2,true) AS (unschedule BOOLEAN);'
                    USING connstr,sql
                    INTO cronresult;

                RETURN cronresult;
            END
            $fn$ LANGUAGE PLPGSQL;

            RAISE NOTICE 'Could not create extension pg_cron. Simulating it instead';
END;
$$ language plpgsql;


COMMIT;