BEGIN;


CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "hstore";


CREATE SCHEMA IF NOT EXISTS auth;


CREATE TABLE IF NOT EXISTS auth.user
(
    id               UUID                              DEFAULT uuid_generate_v4() PRIMARY KEY,
    created_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    login            TEXT                     NOT NULL UNIQUE,
    password_hash    TEXT                     NOT NULL DEFAULT '',

    name_firstname   TEXT                     NOT NULL DEFAULT '',
    name_lastname    TEXT                     NOT NULL DEFAULT '',
    name_displayname TEXT                     NOT NULL DEFAULT '',
    birthday         DATE                              DEFAULT NULL,
    locale           TEXT                     NOT NULL DEFAULT '',
    time_zone        TEXT                     NOT NULL DEFAULT '',
    picture_url      TEXT                     NOT NULL DEFAULT '',
    profile          HSTORE                   NOT NULL DEFAULT '',

    verified_at_utc  TIMESTAMP WITH TIME ZONE          DEFAULT NULL,
    blocked_at_utc   TIMESTAMP WITH TIME ZONE          DEFAULT NULL,
    superuser_at_utc TIMESTAMP WITH TIME ZONE          DEFAULT NULL
);

SELECT enable_automatic_updated_at('auth.user');


CREATE TABLE IF NOT EXISTS auth.user_verification
(
    token           UUID PRIMARY KEY,
    user_id         UUID                     NOT NULL REFERENCES auth.user (id) ON UPDATE CASCADE ON DELETE CASCADE,
    valid_until_utc TIMESTAMP WITH TIME ZONE NOT NULL,

    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

SELECT enable_automatic_updated_at('auth.user_verification');


CREATE TABLE IF NOT EXISTS auth.session
(
    key            BYTEA PRIMARY KEY,
    data           BYTEA                    NOT NULL DEFAULT '',
    expires_at_utc TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    user_id        UUID                              DEFAULT NULL REFERENCES auth.user (id) ON UPDATE CASCADE On DELETE CASCADE,
    user_agent     TEXT                     NOT NULL DEFAULT '',

    created_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
--CREATE INDEX IF NOT EXISTS http_sessions_expiry_idx ON http_sessions (expires_on);
--CREATE INDEX IF NOT EXISTS http_sessions_key_idx ON http_sessions (key);
SELECT enable_automatic_updated_at('auth.session');


COMMIT;