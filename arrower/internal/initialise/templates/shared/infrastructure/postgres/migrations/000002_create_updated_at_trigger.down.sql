BEGIN;


DROP FUNCTION IF EXISTS set_updated_at();
DROP FUNCTION IF EXISTS enable_automatic_updated_at(_tbl regclass);


COMMIT;