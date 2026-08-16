-- +goose Up
--
-- Seed data, converted from the assessment's initial dataset.
--
-- Kept OUT of 001_schema.sql on purpose: schema and data have different
-- lifecycles. Tests and a production deploy want the schema without eleven
-- fictional laptops, and a migration that mixes both can't give you one
-- without the other.
--
-- SEEDS INVENTORY ONLY. No accounts and no rentals are created here. The
-- source dataset implied two holders (an "assignedTo" address and one unnamed
-- holder), and an earlier revision of this file did create placeholder
-- profiles plus their open rentals to keep `in_use` consistent. That produced
-- a database nobody could unstick: the placeholders had no usable password, so
-- no one could log in as them, and rentals may only be returned by their
-- owner. Two devices were therefore permanently unavailable.
--
-- Seeding inventory alone keeps every seeded row in a state a real user can
-- act on. The only account in the system is the admin created at startup from
-- ADMIN_EMAIL / ADMIN_PASSWORD.
--
-- ============================================================================
-- THE SOURCE DATA IS DELIBERATELY DIRTY. Every deviation below is a decision,
-- not a cleanup, and is marked [D-n] at the point where it applies.
-- ============================================================================
--
-- [D-9] The descriptions below are AUTHORED, not source data — the source
-- carried a description for only three devices. They exist because semantic
-- search embeds name + brand + description, and "Apple iPhone 13 Pro Max /
-- Apple" is too little text to place a device near a query like "something to
-- test a mobile app on". Each one states what the device is and what it is
-- used for, since a query describes a need rather than a model number.
--
-- This does not contradict the "never invent" rule applied to [D-3]: that rule
-- is about not fabricating a missing source *value*. These are catalogue copy
-- about well-known products, clearly the operator's own text, and none of them
-- assert anything about the source record. Where the source did carry a note
-- ([D-2], [D-3], [D-5], [D-6], [D-7]) that note is preserved verbatim as the
-- closing sentence, so the audit trail still reads out of the column.

INSERT INTO hardware (id, name, brand, purchase_date, status, description) VALUES

-- [D-1] Status vocabulary mapped "Available"->available, "In Use"->in_use,
-- "Repair"->repair. Display casing belongs in the UI, not the database.
--
-- [D-6] The two devices the source marked "In Use" are seeded as 'available'.
-- `in_use` without an open rental row violates the model's core invariant, and
-- an open rental requires an account that can log in to close it. Neither
-- holder from the source is a real user of this system, so the honest starting
-- state is "on the shelf". The rental history begins empty.
(1, 'Apple iPhone 13 Pro Max', 'Apple', '2021-11-23', 'available',
    'Apple flagship smartphone from 2021, with a 6.7-inch OLED display, the A15 Bionic chip and a triple rear camera. '
    || 'This is the primary iOS device in the pool and the first thing to reach for when testing how a mobile app looks and behaves on a recent iPhone. '
    || 'It runs current versions of iOS, so it covers both the newest APIs and the largest phone screen the team supports. '
    || 'Front-end engineers use it to check responsive layouts, touch targets and safe-area insets on a notched display. '
    || 'QA use it to verify push notifications, biometric login with Face ID, and camera or photo-library permissions. '
    || 'The cameras also make it the device to take out for product photography, short marketing video and demo recordings. '
    || 'Battery life covers a full day of testing away from a desk. '
    || 'Book it together with the Galaxy S21 whenever a change needs signing off on both mobile platforms.'),
(2, 'Apple MacBook Pro 13', 'Apple', '2021-12-20', 'available',
    'A 13-inch Apple laptop running macOS, issued as a general-purpose development machine. '
    || 'It handles writing and building software, running local servers and databases, and everyday office work such as documents, spreadsheets, email and video calls. '
    || 'Its size and battery life make it a common choice for conference travel, client visits and working away from the office. '
    || 'It is also the machine to book when a desktop application needs compiling, signing or notarising for macOS. '
    || 'Developers use it for iOS work as well, since Xcode and the iOS simulators only run on macOS. '
    || 'The keyboard and trackpad are comfortable for long editing sessions without an external mouse. '
    || 'Ports are limited, so pair it with a dock when several peripherals or external displays are needed. '
    || 'A sensible default request for anyone who needs a portable computer rather than one specific device.'),
(3, 'Razer Basilisk V2', 'Razer', '2021-06-05', 'repair',
    'A wired ergonomic gaming mouse with a high-precision optical sensor, adjustable sensitivity and several programmable buttons. '
    || 'In the office it is issued as a desk peripheral for anyone who prefers a full-size mouse with a thumb rest over a laptop trackpad. '
    || 'The precision suits detailed design, illustration and photo-editing work as much as it does gaming. '
    || 'Programmable buttons can be mapped to shortcuts in design tools or an IDE, which speeds up repetitive editing. '
    || 'Being wired, it never needs charging and introduces no wireless latency or pairing trouble. '
    || 'The shape is right-handed, so it is a poor fit for left-handed users. '
    || 'It is a comfort upgrade rather than a necessity, and is usually requested alongside a laptop. '
    || 'This unit is out of service awaiting repair and cannot be issued until it has been checked.'),
(4, 'SAMSUNG Galaxy S21', 'Samsung', '2021-11-23', 'available',
    'Samsung Android flagship smartphone from 2021, with a 6.2-inch display, a triple rear camera and One UI over Android. '
    || 'This is the primary Android handset in the pool and the counterpart to the iPhone for cross-platform mobile testing. '
    || 'Use it to confirm an app renders and behaves correctly on recent Android versions and on Samsung One UI, where most Android-specific bugs surface. '
    || 'QA use it for push notifications, deep links, back-button behaviour and runtime permission prompts, all of which differ from iOS. '
    || 'Front-end engineers use it to check layouts on a smaller screen than the iPhone 13 Pro Max, which catches cramped spacing. '
    || 'The camera is good enough for product shots when the iPhone is already out on loan. '
    || 'It is compact enough to carry alongside a laptop for testing on the move. '
    || 'Pair it with the iPhone whenever a mobile change needs verifying on both platforms.'),

-- [D-2] Source says "Available" while the note says the battery is swelling
-- and it must not be issued. A swelling lithium cell is a fire hazard, so the
-- note wins and this is forced to 'repair'. This is the one place the
-- migration deliberately contradicts the source.
(5, 'Dell XPS 15 9510', 'Dell', '2022-03-15', 'repair',
    'A 15-inch Windows laptop with a discrete GPU and a high-resolution display. '
    || 'It is issued for development work that needs more screen area than a 13-inch machine, and for light 3D, CAD and video rendering that benefits from dedicated graphics. '
    || 'It is also the main Windows device for checking how web applications look and behave outside macOS, including browser differences and font rendering. '
    || 'Designers borrow it when exporting large assets or working with video timelines. '
    || 'The larger chassis makes it heavier than the MacBooks, so it suits desk work more than travel. '
    || 'Windows-specific QA, such as installer testing or verifying a desktop build, belongs on this machine. '
    || 'It stays in repair until the swollen cell has been replaced and battery health verified. '
    || 'Battery swelling, do not issue without service.'),

-- [D-3] Source purchaseDate '2027-10-10' is in the future. Nulled rather than
-- guessed at: any plausible correction is invention, and invented data is
-- worse than absent data. Raw value preserved in the description.
(6, 'Logitech MX Master 3', 'Logitech', NULL, 'available',
    'A wireless ergonomic mouse built for long stretches of desk work, with a sculpted shape, a horizontal scroll wheel and configurable gesture buttons. '
    || 'It connects over Bluetooth or the bundled USB receiver and pairs with up to three machines at once, switching between them with a button. '
    || 'That suits anyone moving between a laptop and a desktop, or between a work machine and a test machine. '
    || 'The free-spinning scroll wheel is useful for long documents, large spreadsheets and code files. '
    || 'Battery life runs to several weeks per charge over USB-C, so it rarely needs attention. '
    || 'It is normally issued alongside a laptop as a comfort upgrade over the built-in trackpad. '
    || 'Like the Basilisk, the shape is right-handed only. '
    || 'Data quality: source purchase date "2027-10-10" rejected as future-dated.'),

-- [D-6] Source carried "assignedTo": "j.doe@booksy.com" here. The assignment
-- is recorded in the description rather than as a rental, because a rental
-- would need an account that can return it.
(7, 'Sony WH-1000XM4', 'Sony', '2022-01-12', 'available',
    'Over-ear wireless headphones with active noise cancellation and roughly thirty hours of battery life per charge. '
    || 'They are issued mainly for focus work in the open-plan office, where the noise cancelling makes a real difference to concentration. '
    || 'They are equally useful for taking calls somewhere noisy such as a cafe, an airport or a shared workspace. '
    || 'The built-in microphone is good enough for video meetings, remote interviews and recording quick voice notes. '
    || 'They connect over Bluetooth to a laptop or a phone and hold two connections at once, so a call on the phone interrupts laptop audio cleanly. '
    || 'A wired 3.5mm option is included for situations where Bluetooth is not permitted or not available. '
    || 'They fold into a case, which makes them practical to take on work travel. '
    || 'Data quality: source listed this as assigned to j.doe@booksy.com. '
    || 'Seeded as available; re-issue through the app to create a real rental.'),

-- [D-4] Corrections applied only where unambiguous: 'Appel' is a clear
-- misspelling, and '22-05-2023' can only be DD-MM-YYYY since there is no
-- month 22. Contrast with [D-3], where no reading is forced.
(9, 'iPad Pro 12.9', 'Apple', '2023-05-22', 'available',
    'A 12.9-inch iPadOS tablet with a high-refresh ProMotion display and Apple Pencil support. '
    || 'It is used for sketching, wireframing and early design exploration, where drawing directly on the screen is faster than working with a mouse. '
    || 'Designers and product people use it in review sessions to mark up screens and annotate PDFs in front of a stakeholder. '
    || 'It is also the device for checking how a responsive layout behaves at tablet size, a distinct breakpoint from both phone and desktop. '
    || 'Mobile app testing on a large iPadOS screen belongs here too, since split view and multitasking behave differently from an iPhone. '
    || 'The size makes it a comfortable reader for long specifications and research papers. '
    || 'With a keyboard case it doubles as a light machine for writing and email while travelling. '
    || 'It can also act as a portable second display for a laptop when working away from the office.'),

-- [D-5] Source status 'Unknown' is not a value our model has. Quarantined as
-- 'repair' — imperfect, since the device isn't broken but unidentified, yet
-- 'repair' is the only existing state that blocks renting. A fourth status
-- ('quarantined') is the better long-term model and a deliberate deferral.
(10, 'Unknown Device', 'Unknown Brand', NULL, 'repair',
     'An unidentified item with no usable make, model or purchase date on record. '
     || 'Nobody has yet established what it is or whether it still works, so it cannot be '
     || 'described or issued. It is held in repair purely to keep it out of circulation '
     || 'until someone inspects it and files a proper record. '
     || 'Data quality: source status "Unknown"; brand missing. '
     || 'Quarantined pending identification.'),

-- [D-2] Same reasoning as hardware 5: liquid damage, not rentable.
(11, 'MacBook Air M2', 'Apple', '2023-08-01', 'repair',
    'A lightweight 13-inch Apple laptop with the M2 chip, fanless and silent under normal load. '
    || 'It is normally issued to people whose work is email, documents, browsing, video calls and light editing rather than heavy compilation. '
    || 'Battery life comfortably covers a full working day, which makes it the best travel machine in the pool for conferences and client visits. '
    || 'It is thin and light enough to carry all day without a dedicated laptop bag. '
    || 'macOS means it can also run Xcode and the iOS simulator for light mobile work, though builds are slower than on the Pro. '
    || 'Designers use it as a portable machine for presenting work when a larger screen is not needed. '
    || 'The keyboard and screen match the Pro line, so nothing is lost for everyday use. '
    || 'Returned by user with liquid damage. Keyboard sticky.'),

-- [D-7] Source reused id 4, which a PRIMARY KEY cannot allow. First
-- occurrence keeps it; this one is appended at 12 rather than backfilling the
-- unexplained gap at id 8. First-wins is arbitrary but stable and matches
-- source order.
(12, 'Duplicate ID Test Laptop', 'Lenovo', '2023-01-01', 'repair',
     'A Lenovo laptop running Windows, kept as a spare loaner for staff whose primary machine is away being serviced or replaced. '
     || 'It is specified for routine office work: documents, spreadsheets, email, browsing and video calls. '
     || 'It is deliberately not allocated to anyone permanently, so it is usually the fastest device to get hold of at short notice. '
     || 'It is also useful as a clean Windows machine for testing an installer or reproducing a bug on a stock configuration. '
     || 'Onboarding sometimes uses it as a stopgap while a new starter waits for their own equipment. '
     || 'Because it passes between many people, it is wiped and re-imaged between loans rather than carried over. '
     || 'It stays in repair until that re-imaging has been completed. '
     || 'Data quality: source record had duplicate id 4; reassigned to 12.');

-- +goose Down
DELETE FROM hardware WHERE id IN (1, 2, 3, 4, 5, 6, 7, 9, 10, 11, 12);
