\connect postgres

CREATE EXTENSION IF NOT EXISTS pg_cron;

-- Grant usage of the cron schema to regular user
-- The user's name is taken from the (docker-compose set) environment variable `POSTGRES_USER`.

-- Make env variables available as table, see: https://stackoverflow.com/a/64294517
CREATE TEMPORARY TABLE env_tmp
(
    e text
);
CREATE TEMPORARY TABLE env
(
    k text,
    v text
);

COPY env_tmp (e) FROM PROGRAM 'env';

INSERT INTO env
SELECT (regexp_split_to_array(e, '={1,1}'))[1],
       (regexp_match(e, '={1,1}(.*)'))[1]
FROM env_tmp;

DO
$$
    DECLARE
        username TEXT;
    BEGIN
        -- take the user name given in env variable: POSTGRES_USER
        SELECT v FROM env WHERE k = 'POSTGRES_USER' INTO username;

        -- grant usage of the cron schema to regular user
        --EXECUTE FORMAT('GRANT USAGE ON SCHEMA cron TO %I;', username);
    END
$$ LANGUAGE PLPGSQL;