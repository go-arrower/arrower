BEGIN;


CREATE SCHEMA admin;
CREATE TABLE admin.setting
(
    setting TEXT PRIMARY KEY,
    value   TEXT NOT NULL DEFAULT ''
);

CREATE TABLE "some_table"
(
    name TEXT PRIMARY KEY
);

CREATE TABLE "other_table"
(
    name TEXT PRIMARY KEY
);

COMMIT;