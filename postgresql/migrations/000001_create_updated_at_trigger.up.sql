BEGIN;


CREATE OR REPLACE FUNCTION enable_automatic_updated_at(_tbl regclass) RETURNS VOID AS $$
BEGIN
    EXECUTE FORMAT('CREATE TRIGGER set_updated_at BEFORE UPDATE ON %s FOR EACH ROW EXECUTE PROCEDURE set_updated_at()',
        _tbl);
END;
$$ LANGUAGE plpgsql;


CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    IF (
        NEW IS DISTINCT FROM OLD AND
        NEW.updated_at IS NOT DISTINCT FROM OLD.updated_at
    ) THEN
        NEW.updated_at := current_timestamp;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


COMMIT;