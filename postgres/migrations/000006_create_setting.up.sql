BEGIN;


CREATE TABLE IF NOT EXISTS arrower.setting
(
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    key        TEXT                     NOT NULL DEFAULT '',
    value      TEXT                     NOT NULL DEFAULT '',

    PRIMARY KEY (key)
);

SELECT enable_automatic_updated_at('arrower.setting');


COMMIT;