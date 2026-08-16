-- +goose Up

-- [D-8] The deciding reason is the updated_at trigger: it fires on any UPDATE
-- that does not set updated_at itself, so writing a recomputed vector into a
-- hardware column would mark the device as modified when only a derived
-- artifact changed. Separating them also makes switching embedding models a
-- DELETE rather than a migration, and leaves somewhere to record which model
-- produced a vector and which text it was produced from.
CREATE TABLE IF NOT EXISTS hardware_embeddings (
    hardware_id INTEGER PRIMARY KEY REFERENCES hardware (id) ON DELETE CASCADE,
    model       TEXT    NOT NULL,
    dimensions  INTEGER NOT NULL CHECK (dimensions > 0),
    source_hash TEXT    NOT NULL,
    vector      BLOB    NOT NULL CHECK (length(vector) = dimensions * 4),
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS hardware_embeddings_set_updated_at
AFTER UPDATE ON hardware_embeddings
FOR EACH ROW WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE hardware_embeddings
       SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
     WHERE hardware_id = NEW.hardware_id;
END;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS hardware_embeddings;
