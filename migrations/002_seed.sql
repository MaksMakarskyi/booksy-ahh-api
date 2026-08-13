-- +goose Up
--
-- Seed data, converted from the assessment's initial dataset.
--
-- Kept OUT of 001_schema.sql on purpose: schema and data have different
-- lifecycles. Tests and a production deploy want the schema without eleven
-- fictional laptops, and a migration that mixes both can't give you one
-- without the other.
--
-- ============================================================================
-- THE SOURCE DATA IS DELIBERATELY DIRTY. Every deviation below is a decision,
-- not a cleanup, and is marked [D-n] at the point where it applies.
-- ============================================================================

-- --- Accounts needed for referential integrity --------------------------
--
-- password_hash '!' is a sentinel meaning "no password set, login disabled".
-- No credentials are seeded: the admin account is bootstrapped at startup
-- from ADMIN_EMAIL/ADMIN_PASSWORD so a known password never ships in git.

INSERT INTO profiles (id, email, full_name, role, password_hash) VALUES
    (1, 'j.doe@booksy.com', 'J. Doe', 'employee', '!'),
    -- [D-6] Placeholder holder for hardware 2. Not a real person; the
    -- .invalid TLD (RFC 2606) guarantees it can never receive mail.
    (2, 'unknown-holder@hardware-hub.invalid', 'Unknown holder (migrated)',
        'employee', '!');

-- --- Inventory ----------------------------------------------------------

INSERT INTO hardware (id, name, brand, purchase_date, status, description) VALUES

-- [D-1] Status vocabulary mapped "Available"->available, "In Use"->in_use,
-- "Repair"->repair. Display casing belongs in the UI, not the database.
(1, 'Apple iPhone 13 Pro Max', 'Apple',   '2021-11-23', 'available', NULL),
(2, 'Apple MacBook Pro 13',    'Apple',   '2021-12-20', 'in_use',    NULL),
(3, 'Razer Basilisk V2',       'Razer',   '2021-06-05', 'repair',    NULL),
(4, 'SAMSUNG Galaxy S21',      'Samsung', '2021-11-23', 'available', NULL),

-- [D-2] Source says "Available" while the note says the battery is swelling
-- and it must not be issued. A swelling lithium cell is a fire hazard, so the
-- note wins and this is forced to 'repair'. This is the one place the
-- migration deliberately contradicts the source.
(5, 'Dell XPS 15 9510', 'Dell', '2022-03-15', 'repair',
    'Battery swelling, do not issue without service.'),

-- [D-3] Source purchaseDate '2027-10-10' is in the future. Nulled rather than
-- guessed at: any plausible correction is invention, and invented data is
-- worse than absent data. Raw value preserved in the description.
(6, 'Logitech MX Master 3', 'Logitech', NULL, 'available',
    'Data quality: source purchase date "2027-10-10" rejected as future-dated.'),

(7, 'Sony WH-1000XM4', 'Sony', '2022-01-12', 'in_use', NULL),

-- [D-4] Corrections applied only where unambiguous: 'Appel' is a clear
-- misspelling, and '22-05-2023' can only be DD-MM-YYYY since there is no
-- month 22. Contrast with [D-3], where no reading is forced.
(9, 'iPad Pro 12.9', 'Apple', '2023-05-22', 'available', NULL),

-- [D-5] Source status 'Unknown' is not a value our model has. Quarantined as
-- 'repair' — imperfect, since the device isn't broken but unidentified, yet
-- 'repair' is the only existing state that blocks renting. A fourth status
-- ('quarantined') is the better long-term model and a deliberate deferral.
(10, 'Unknown Device', '', NULL, 'repair',
     'Data quality: source status "Unknown"; brand missing. '
     || 'Quarantined pending identification.'),

-- [D-2] Same reasoning as hardware 5: liquid damage, not rentable.
(11, 'MacBook Air M2', 'Apple', '2023-08-01', 'repair',
     'Returned by user with liquid damage. Keyboard sticky.'),

-- [D-7] Source reused id 4, which a PRIMARY KEY cannot allow. First
-- occurrence keeps it; this one is appended at 12 rather than backfilling the
-- unexplained gap at id 8. First-wins is arbitrary but stable and matches
-- source order.
(12, 'Duplicate ID Test Laptop', 'Lenovo', '2023-01-01', 'repair',
     'Data quality: source record had duplicate id 4; reassigned to 12.');

-- --- Open rentals -------------------------------------------------------
--
-- [D-6] Every 'in_use' device MUST have an open rental row.

INSERT INTO rentals (hardware_id, user_id, rented_at, returned_at) VALUES
    -- Source gave "assignedTo": "j.doe@booksy.com".
    (7, 1, '2022-01-12T09:00:00Z', NULL),
    (2, 2, '2021-12-20T09:00:00Z', NULL);

-- +goose Down
DELETE FROM rentals  WHERE hardware_id IN (2, 7);
DELETE FROM hardware WHERE id IN (1, 2, 3, 4, 5, 6, 7, 9, 10, 11, 12);
DELETE FROM profiles WHERE id IN (1, 2);
