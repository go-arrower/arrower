BEGIN;


--
-- This is a workaround, as pg_cron can only be installed and used in one database.
-- It is assumed that pg_cron is installed in the default database 'postgres' and this user has access to it.
-- The following will simulate pg_cron as if it was installed in this database. The main purpose is to run automated arrower integration tests.
--
-- CREATE EXTENSION IF NOT EXISTS pg_cron;
-- (details: ERROR: can only create extension in database postgres (SQLSTATE P0001))


CREATE SCHEMA IF NOT EXISTS cron;

CREATE EXTENSION IF NOT EXISTS dblink;

-- EXECUTE FORMAT has to be used:
-- Variable substitution currently works only in SELECT, INSERT, UPDATE, and DELETE commands, because the main SQL
-- engine allows query parameters only in these commands. To use a non-constant name or value in other statement types
-- (generically called utility statements), you must construct the utility statement as a string and EXECUTE it.
-- See: https://www.postgresql.org/docs/current/plpgsql-implementation.html#PLPGSQL-VAR-SUBST
-- DO $$
-- BEGIN
-- EXECUTE FORMAT($createView$
-- CREATE VIEW cron.job AS
-- (
--     SELECT *
--     FROM dblink('dbname=postgres user=arrower', %L) AS job
--         (
--           jobid BIGINT,
--           schedule TEXT,
--           command TEXT,
--           nodename TEXT,
--           nodeport INTEGER,
--           database TEXT,
--           username TEXT,
--           active BOOLEAN,
--           jobname TEXT
--         )
-- );$createView$,
--     (SELECT FORMAT('SELECT * FROM cron.job WHERE database=%L;', CURRENT_DATABASE())));
-- END
-- $$ LANGUAGE plpgsql;

-- CREATE OR REPLACE SERVER l FOREIGN DATA WRAPPER postgres_fdw OPTIONS (dbname 'postgres');
--
-- INSERT INTO cron.job (command, jobname)
-- VALUES('San Jose', 'insert-some-name-via-view');
-- ERROR:  Views that do not select from a single table or view are not automatically updatable.cannot insert into view "job"
-- ERROR:  cannot insert into view "job"
--                    SQL state: 55000
--                    Detail: Views that do not select from a single table or view are not automatically updatable.
-- Hint: To enable inserting into the view, provide an INSTEAD OF INSERT trigger or an unconditional ON INSERT DO INSTEAD rule.




-- dblink does not allow write access to the view... so use fdw
-- consequence: pgAdmin will show the view as read only, but it can be inserted into and updated via SQL
CREATE EXTENSION IF NOT EXISTS postgres_fdw;

CREATE SERVER IF NOT EXISTS localhost_postgres_db FOREIGN DATA WRAPPER postgres_fdw OPTIONS (dbname 'postgres');

--create user mapping IF NOT EXISTS FOR CURRENT_USER SERVER localhost_postgres_db OPTIONS (user 'arrower');
DO $$
    BEGIN
        EXECUTE FORMAT($createView$create user mapping IF NOT EXISTS FOR CURRENT_USER SERVER localhost_postgres_db OPTIONS (user '%s');$createView$, CURRENT_USER);
    END
$$ LANGUAGE plpgsql;

IMPORT FOREIGN SCHEMA cron
    FROM SERVER localhost_postgres_db INTO cron;
-- TODO pull only view of tables that filters for current dbname and not others?
-- TODO: does this also import functions (?) that would be amazing => no


-- create foreign table cron.job (
--     jobid BIGINT,
--     schedule TEXT,
--     command TEXT,
--     nodename TEXT,
--     nodeport INTEGER,
--     database TEXT,
--     username TEXT,
--     active BOOLEAN,
--     jobname TEXT
--     ) server l options(table_name 'job') ;



-- DO $$
--     BEGIN
--         EXECUTE FORMAT($createView$
-- CREATE VIEW cron.job AS
-- (
--     SELECT *
--     FROM dblink('dbname=postgres user=arrower', %L) AS job
--         (
--           jobid BIGINT,
--           schedule TEXT,
--           command TEXT,
--           nodename TEXT,
--           nodeport INTEGER,
--           database TEXT,
--           username TEXT,
--           active BOOLEAN,
--           jobname TEXT
--         )
-- );$createView$,
--                        (SELECT FORMAT('SELECT * FROM cron.job WHERE database=%L;', CURRENT_DATABASE())));
--     END
-- $$ LANGUAGE plpgsql;

CREATE EXTENSION IF NOT EXISTS dblink;


-- TODO: how to test this to make sure it works and stays compatible with pg_cron changes in behaviour or API?
-- exec the tutorial commands again postgres db and compare results with exec them against test db (?)



-- TODO check if username and db should be allowed as param or always forced to be the current user (?) Both have good points
CREATE OR REPLACE FUNCTION cron.alter_job(job_id BIGINT, schedule TEXT, command TEXT, database TEXT, username TEXT, active BOOLEAN) RETURNS VOID AS $$
DECLARE
    sql        TEXT;
    connstr    NAME;
BEGIN
    connstr := 'dbname=postgres user=' || CURRENT_USER;

    sql := FORMAT('SELECT cron.alter_job(%L,%L,%L,%L,%L,%L);', job_id, schedule, command, database, username, active);
    EXECUTE 'SELECT * FROM dblink($1,$2,true) AS (schedule BIGINT);'
        USING connstr,sql;
END
$$ LANGUAGE PLPGSQL;


CREATE OR REPLACE FUNCTION cron.schedule(job_name TEXT, schedule TEXT, command TEXT) RETURNS BIGINT AS $$
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
$$ LANGUAGE PLPGSQL;


CREATE OR REPLACE FUNCTION cron.schedule(schedule TEXT, command TEXT) RETURNS BIGINT AS $$
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
$$ LANGUAGE PLPGSQL;


CREATE OR REPLACE FUNCTION cron.schedule_in_database(job_name TEXT, schedule TEXT, command TEXT, database TEXT, username TEXT, active BOOLEAN) RETURNS BIGINT AS $$
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
$$ LANGUAGE PLPGSQL;


CREATE OR REPLACE FUNCTION cron.unschedule(job_id BIGINT) RETURNS BOOLEAN AS $$
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
$$ LANGUAGE PLPGSQL;


CREATE OR REPLACE FUNCTION cron.unschedule(job_name TEXT) RETURNS BOOLEAN AS $$
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
$$ LANGUAGE PLPGSQL;


COMMIT;