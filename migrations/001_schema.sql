-- +goose Up
CREATE TABLE IF NOT EXISTS hardware (
    id            INTEGER PRIMARY KEY,
    name          TEXT NOT NULL,
    brand         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    purchase_date TEXT,
    status        TEXT NOT NULL DEFAULT 'available'
                  CHECK (status IN ('available', 'in_use', 'repair')),
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS hardware_status_idx ON hardware (status);
CREATE INDEX IF NOT EXISTS hardware_brand_idx  ON hardware (brand);

CREATE TABLE IF NOT EXISTS profiles (
    id            INTEGER PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    full_name     TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'employee'
                  CHECK (role IN ('employee', 'admin')),
    password_hash TEXT NOT NULL,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS rentals (
    id          INTEGER PRIMARY KEY,
    hardware_id INTEGER NOT NULL REFERENCES hardware (id) ON DELETE CASCADE,
    user_id     INTEGER NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    rented_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    returned_at TEXT,

    CHECK (returned_at IS NULL OR returned_at >= rented_at)
);

CREATE UNIQUE INDEX IF NOT EXISTS rentals_one_active_per_hardware_idx
    ON rentals (hardware_id) WHERE returned_at IS NULL;

CREATE INDEX IF NOT EXISTS rentals_user_id_idx ON rentals (user_id);

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS hardware_set_updated_at
AFTER UPDATE ON hardware
FOR EACH ROW WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE hardware
       SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
     WHERE id = NEW.id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS profiles_set_updated_at
AFTER UPDATE ON profiles
FOR EACH ROW WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE profiles
       SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
     WHERE id = NEW.id;
END;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS rentals;
DROP TABLE IF EXISTS profiles;
DROP TABLE IF EXISTS hardware;
