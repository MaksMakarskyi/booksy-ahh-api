# Hardware Hub — API

Internal tool for Booksy employees to manage, rent and maintain company equipment.

This repository contains the **backend API**, written in Go with a SQLite database.

- **Live demo:** `<DEPLOYMENT_URL>` *(to be filled after deployment)*
- **Frontend repository:** `<FRONTEND_REPO_URL>` *(to be filled)*

## Table of contents

- [Stack and why](#stack-and-why)
- [Quick start](#quick-start)
- [Configuration](#configuration)
- [API reference](#api-reference)
- [Architecture](#architecture)
- [Data model](#data-model)
- [Implementation status and trade-offs](#implementation-status-and-trade-offs)
- [The AI development log](#the-ai-development-log)
- [Deployment](#deployment)

## Stack and why

| Choice | Reason |
| --- | --- |
| **Go 1.26** | The task allows any language in which you are more productive. Go gives a single static binary, which makes deployment and review trivial. Moreover, Go is a compiled language, which gives some extra type safety for the application. Also, the language allows you to implement advanced concurrency patterns using goroutines and channels, which are features no other language gives us. Go brings other advantages as well, including smaller image size and better performance according to numerous articles and case studies written by employees from Big Tech companies that you can find on the internet. So, from my personal point of view, Go is a better choice for the long run to some extent than something like Python. Therefore, I decided to pick Go for this backend. |
| **SQLite** (`modernc.org/sqlite`) | Suggested by the task. Pure-Go driver, so `CGO_ENABLED=0` produces a static binary with no libc dependency. SQLite was chosen mainly because it allows a seamless testing experience, though I would definitely pick something like Postgres for a real production app, since SQLite is quite limited. For example, it does not allow creating enumerators and stores dates as strings, which removes some database-level checks and may result in invalid DB states if a developer is not careful and does not ensure those checks at the application level. |
| **goose** | Versioned SQL migrations, embedded into the binary so the app self-migrates on boot. |
| **scany** | Maps SQL rows onto structs without an ORM. Queries stay hand-written and visible. |
| **go-playground/validator** | Declarative request validation, extended with two custom rules. |
| **bcrypt** + **golang-jwt/v5** | Password hashing and HS256 access tokens. |

## Quick start

### Option A — Docker (recommended for review)

```bash
cp .env.example .env     # then fill in the values, see Configuration
make compose
```

The API is available on `http://localhost:8080`.

The container is self-sufficient: migrations are **embedded in the binary** and applied at startup, and the first admin account is created from `ADMIN_EMAIL` / `ADMIN_PASSWORD`. No manual migration step, no seeding step.

### Option B — local Go toolchain

```bash
cp .env.example .env     # then fill in the values
make run
```

Requires Go 1.26+. Nothing else — no database server to install.

### Verify it works

```bash
curl -s localhost:8080/healthz

TOKEN=$(curl -s -X POST localhost:8080/auth/token \
  -H 'Content-Type: application/json' \
  -d '{"email":"<ADMIN_EMAIL>","password":"<ADMIN_PASSWORD>"}' \
  | jq -r .data.access_token)

curl -s localhost:8080/hardware -H "Authorization: Bearer $TOKEN" | jq
```

### Make targets

| Target | Description |
| --- | --- |
| `make run` | Run the API locally |
| `make test` | Run the test suite |
| `make format` | `go fmt ./...` |
| `make tidy` | `go mod tidy` |
| `make compose` | Build and run in Docker |
| `make migrate` | Apply migrations with the goose CLI |
| `make migrate-down` | Roll back one migration |
| `make migrate-status` | Show applied migrations |
| `make migrate-reset` | Drop everything and re-apply (destroys data) |

The `make migrate*` targets exist for development. In normal operation the binary migrates itself.

## Configuration

All configuration is environment variables. `.env` is loaded by `make` targets and by Docker Compose; in production the values come from the environment directly. See `.env.example`.

| Variable | Required | Default | Notes |
| --- | --- | --- | --- |
| `APP_ENV` | no | `production` | `development` \| `production`. Controls CORS and whether errors carry a `debug` field. |
| `PORT` | no | `8080` | |
| `DATABASE_URL` | **yes** | — | SQLite DSN. The pragmas are not optional, see below. |
| `JWT_SECRET` | **yes** | — | HS256 signing key. Rejected at startup if under 32 bytes. |
| `JWT_TTL` | no | `12h` | Access token lifetime. |
| `CORS_ORIGINS` | no | `*` | Comma-separated. Set to the frontend origin in production. |
| `ADMIN_EMAIL` | **yes** | — | First admin, created at startup if absent. |
| `ADMIN_PASSWORD` | **yes** | — | Must satisfy the same password policy as the API. |
| `ADMIN_NAME` | no | `Administrator` | |
| `GOOSE_TABLE` | no | `goose_migrations` | Migration bookkeeping table. |
| `GOOSE_DRIVER`, `GOOSE_DBSTRING`, `GOOSE_MIGRATION_DIR` | no | — | Only used by the goose **CLI** (`make migrate`). |

### The database DSN

```
file:data/hardware-hub.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)
```

Each pragma is load-bearing:

- **`foreign_keys(1)`** — SQLite disables foreign-key enforcement **by default, per connection**. Without it every `REFERENCES` clause in the schema is decorative and orphan rows insert silently.
- **`journal_mode(WAL)`** — readers do not block the writer.
- **`busy_timeout(5000)`** — wait for a lock instead of failing immediately.

The connection pool is capped at **one connection** (`SetMaxOpenConns(1)`). SQLite permits a single writer; serialising all access removes an entire class of `SQLITE_BUSY` errors at a throughput cost this application will never notice.

## API reference

All responses are wrapped in an envelope: `{"data": ...}` on success, `{"error": {...}}` on failure.

Authentication is `Authorization: Bearer <token>` on every route except `/healthz` and `/auth/token`.

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/healthz` | — | Liveness probe |
| `POST` | `/auth/token` | — | Exchange email + password for an access token |
| `GET` | `/hardware` | user | List all equipment |
| `POST` | `/hardware` | **admin** | Add equipment |
| `PATCH` | `/hardware/:id` | **admin** | Edit name / brand / description / purchase date |
| `DELETE` | `/hardware/:id` | **admin** | Remove equipment |
| `PATCH` | `/hardware/:id/repair` | **admin** | Send to repair (`available` → `repair`) |
| `PATCH` | `/hardware/:id/available` | **admin** | Return to stock (`repair` → `available`) |
| `GET` | `/rentals` | user | List **your own** rentals |
| `POST` | `/rentals` | user | Rent a device |
| `PATCH` | `/rentals/:id/return` | user | Return **your own** rental |
| `GET` | `/profiles` | **admin** | List accounts |
| `POST` | `/profiles` | **admin** | Create an account |
| `DELETE` | `/profiles/:id` | **admin** | Remove an account |

### Error format

Validation failures report every bad field at once, keyed by the JSON field name:

```json
{
  "error": {
    "message": "The request payload is invalid.",
    "fields": [
      { "field": "email", "rule": "email", "message": "must be a valid email address" },
      { "field": "password", "rule": "min", "message": "must be at least 8 characters" }
    ],
    "request_id": "GtFBxWlCdxSYSwXrbMxYPaknIQiMcrvK"
  }
}
```

Every response carries an `X-Request-Id`, echoed in the body and in every log line for that request, so a reported failure can be found in the logs.

| Status | When |
| --- | --- |
| `400` | Malformed body, unknown JSON field, failed validation, bad path parameter |
| `401` | Missing / invalid / expired token, wrong credentials |
| `403` | Authenticated but not permitted (non-admin, or another user's rental) |
| `404` | Resource does not exist |
| `409` | Impossible state transition (renting a device in repair, double return, duplicate email) |
| `503` | Request exceeded the 15s timeout |
| `500` | Our bug. The client message is always generic; the full chain is logged. |

### Business rules

- A device can only be rented when `available`. The guard is a compare-and-swap (`UPDATE ... WHERE id = ? AND status = 'available'`), so two simultaneous renters cannot both win.
- Renting flips the status **and** opens a rental row **in one transaction** — the two can never disagree.
- A partial unique index (`rentals_one_active_per_hardware_idx`) enforces at most one open rental per device at the storage layer, independent of application code.
- A rented device cannot be sent to repair; it must be returned first, otherwise the open rental would be orphaned.
- Users list and return only their own rentals, **regardless of role**.
- Accounts cannot self-register. Only an admin creates accounts.

## Architecture

```
cmd/api/                 entrypoint: config → dependencies → migrate → admin → serve
internal/
  auth/                  login, JWT middleware, current-user context
  hardware/              equipment CRUD and repair transitions
  rentals/               rent / return / list
  profiles/              account management + startup admin bootstrap
    roles/               shared role vocabulary (leaf package)
  server/                router, CORS
    config/              environment configuration
    dependencies/        dependency registry
  utils/                 errors, validation, jwt, password, migrate
migrations/              embedded SQL migrations
```

Every domain package has the same five files: `build.go` (composition root), `handler.go` (HTTP), `store.go` (SQL), `models.go`, `requests.go`.

**Layering.** `utils/` and `roles/` are leaves that import nothing internal. Domain packages import `dependencies`, so anything the registry holds must sit *below* the domain — this is why the `jwt` package deals in plain strings rather than domain types, and why `roles` is its own package. Two import cycles were resolved this way during development.

**The store interface.** Each domain declares a `Store` interface next to the handler that consumes it, with `SQLiteStore` implementing it. Handlers depend on the interface, not the implementation, so swapping the database means writing one adapter — and tests can substitute an in-memory fake with no database at all.

**Error handling.** Handlers never choose status codes. They wrap errors with context (`fmt.Errorf("...: %w", err)`) and return them; a single `HTTPErrorHandler` maps sentinel errors (`ErrStoreNotFound`, `ErrStoreConflict`, `ErrStoreForbidden`, ...) onto status codes, logs the full chain, and returns a client-safe message. 5xx responses never leak internals; the detail is in the logs, correlated by request id.

**Validation.** Request structs carry `validate` tags. Two custom rules are registered: `notfuture` (a date not in the future) and `password` (character-class policy). Payload types can opt into two interfaces — `Normalizer` (trim/lowercase before validation) and `SelfValidator` (rules spanning several fields, such as "a PATCH must change something").

**Strict JSON.** The JSON deserialiser is replaced with one that sets `DisallowUnknownFields`. A typo like `{"stauts": ...}` returns 400 instead of being silently ignored, and an attempt to set a field the client does not own (`{"status": "available"}`) is visible rather than quietly dropped.

## Data model

```
profiles                    hardware                     rentals
--------                    --------                     -------
id                          id                           id
email          UNIQUE       name                         hardware_id  → hardware (CASCADE)
full_name                   brand                        user_id      → profiles (RESTRICT)
role           CHECK        description                  rented_at
password_hash               purchase_date                returned_at
created_at                  status        CHECK
updated_at                  created_at
                            updated_at
```

- `hardware.status` ∈ `available | in_use | repair`, `profiles.role` ∈ `employee | admin`, both enforced by `CHECK` constraints (SQLite has no `ENUM`).
- `rentals` is append-only history; a return sets `returned_at` rather than deleting the row. `hardware.status` is a fast projection of that history.
- Timestamps are TEXT in RFC 3339 UTC. Dates are TEXT `YYYY-MM-DD`. SQLite has no date type, and ISO-8601 sorts correctly as a string.
- `ON DELETE CASCADE` on `hardware_id` (deleting a device removes its history), `RESTRICT` on `user_id` (an account with rental history cannot be deleted).
- `updated_at` is maintained by triggers, since SQLite has no `ON UPDATE CURRENT_TIMESTAMP`.

## Implementation status and trade-offs

### ✅ Fully implemented

**The management engine**
- Admin command centre: add / edit / delete equipment, toggle repair status, create and remove accounts.
- Login issuing HS256 JWTs. Accounts are created only by an admin; there is no self-registration.
- Equipment list with name, brand, purchase date and status.

**The rental engine**
- Rent and return with guards against every impossible state: renting a device that is in repair or already out, returning a device twice, returning someone else's rental, sending a rented device to repair.
- Concurrency-safe: compare-and-swap plus a partial unique index, so the invariant holds even if the application has a bug.

**Cross-cutting**
- Role-based authorisation (`employee` / `admin`) from a signed token claim, re-validated on every request.
- Centralised error handling, structured logging with request-id correlation, strict JSON decoding, per-field validation errors.
- Self-migrating binary and startup admin bootstrap, both idempotent.

### ⚡ Shortcuts and "hacks"

**1. Filtering and sorting are done on the client, not the API.**
- *Why acceptable:* the dataset is ~220 bytes per row — a thousand devices is ~54 KB gzipped, fetched once. Client-side filtering is instant, needs no round-trip, and removes dynamic `WHERE`/`ORDER BY` assembly (and its injection surface) from the backend entirely.
- *The threshold:* this holds **until pagination is required**. Filtering and pagination must live on the same side; paginating without moving the filters would silently filter only the loaded page.
- *The future:* `GetAll(ctx)` becomes `List(ctx, filters)`. Because handlers depend on the `Store` interface, the change is confined to one store and one handler.

**2. Access tokens are long-lived (12h) with no refresh flow.**
- *Why acceptable:* an internal tool with a single-day session expectation. A refresh flow is meaningful complexity for no benefit at this scale.
- *The future:* short-lived (15 min) access token plus a rotating refresh token in an httpOnly cookie.

**3. The token is stored in `localStorage` on the frontend.**
- *Why acceptable:* the frontend and API are on different origins, so a cookie would need `SameSite=None; Secure` plus credentialed CORS. `localStorage` + `Authorization` header is the pragmatic choice, and the exposure is bounded by the token lifetime.
- *The risk, stated plainly:* `localStorage` is readable by any injected script, so an XSS becomes a token theft.
- *The future:* httpOnly refresh cookie plus an in-memory access token.

**4. `ON DELETE RESTRICT` makes some accounts undeletable.**
- An employee with any rental history — even fully returned — cannot be deleted. This returns a clear 409 rather than a crash, but it means offboarding needs either an archive flag or a history reassignment step. Deliberate: silently deleting audit history is worse.

**5. Deleting a device cascades its rental history.**
- The inverse choice from the above. Rental history has no meaning without the device it refers to.

### ⚠ Partial / missing

- **Tests.** *(To be completed — see the note at the end of this section.)*
- **AI layer.** Not implemented. Semantic search is the intended feature; the store interface is the seam it would attach to.
- **PATCH cannot clear a nullable field.** `*string` cannot distinguish "absent" from "explicit null" — both decode to `nil`, and `COALESCE` reads `nil` as "keep". So `description` can be set and changed, never removed. Fixing it needs an `Optional[T]` wrapper that records key presence.
- **No pagination.** Deliberate, per shortcut 1.
- **`GET /profiles` returns employees plus the caller**, not other admins. Intentional for an "assignable people" list; arguably surprising for an endpoint named "list accounts".
- **No rate limiting** on `/auth/token`. Echo ships a rate-limiter middleware; it is one line and was left out only for scope.
- **Frontend.** Lives in a separate repository *(link to be added)*.

### 🔮 Next steps — the 24-hour roadmap

1. **Tests.** The rental engine deserves them most: double-rent, wrong-returner, repair-while-rented, and the JWT forgery cases (`alg: none`, foreign secret, tampered payload). The `Store` interface makes these fast — an in-memory fake, no database.
2. **The AI layer — semantic search.** Embed each device's name, brand and description at write time, store the vectors alongside the rows, and rank by cosine similarity at query time. `"something to test a mobile app on"` should return the iPhone and the Galaxy.
3. **Refresh tokens and rate limiting.** The two security items above: shorten the access token, add rotation, and put a limiter on the login endpoint.

## The AI development log

> **Note for the author:** the sections below are pre-filled from the actual development session and are accurate as far as they go. `<...>` placeholders and the prompt trail still need your input.

### Tooling

| Tool | Used for |
| --- | --- |
| `<TOOL_1, e.g. Claude Code>` | `<what you used it for>` |
| `<TOOL_2, e.g. Cursor>` | `<what you used it for>` |
| `<TOOL_3>` | `<...>` |

### Data strategy — auditing the seed

The provided dataset is deliberately dirty. Every record was audited before insertion and each deviation is recorded as a numbered decision in `migrations/002_seed.sql`, with the reasoning in the file itself. Summary:

| Issue in source | Decision |
| --- | --- |
| Duplicate `id: 4` (Galaxy S21 and "Duplicate ID Test Laptop") | A primary key cannot hold both. First occurrence keeps the id; the second is appended at 12 rather than backfilling the unexplained gap at id 8. |
| Brand `"Appel"` | Corrected to `"Apple"` — an unambiguous misspelling. |
| Date `"22-05-2023"` (DD-MM-YYYY) | Parsed to `2023-05-22`. Unambiguous, since there is no month 22. |
| Date `"2027-10-10"` (future) | **Nulled, not guessed.** Any plausible correction is invention, and invented data is worse than absent data. The raw value is preserved in the description. |
| Status `"Unknown"` | Not a value the model has. Quarantined as `repair` so the device cannot be rented until a human identifies it. A fourth status is the better long-term model — a deliberate deferral. |
| Dell XPS: `"Available"` **with** the note *"Battery swelling, do not issue without service."* | **The note wins.** Forced to `repair`. Trusting the status field would let the app hand an employee a fire hazard. This is the one place the migration deliberately contradicts its source. |
| MacBook Air M2: `"Available"` with liquid damage in `history` | Same reasoning — forced to `repair`. |
| `assignedTo: "j.doe@booksy.com"` on an in-use device | Became a real `profiles` row plus an open `rentals` row, because the model requires every `in_use` device to have one. |
| `"In Use"` with **no** assignee | The most debatable call. Marking it available would claim it is on the shelf when someone physically holds it; inventing an owner would fabricate an audit trail. Resolved with a clearly-labelled placeholder account on the `.invalid` TLD (RFC 2606), so the device stays correctly unavailable and the unknown holder is visible rather than hidden. |
| Empty brand, null date | Kept as-is. Absent is a legitimate value. |

The general principle: **normalise what is unambiguous, quarantine what is not, and never invent.**

### Prompt trail

`<LINK_TO_PROMPT_HISTORY or PROMPTS.md>`

`<Add the prompts that shaped the architecture: the stack decision, the route design discussion, the store-interface pattern.>`

### The "correction"

AI assistance produced several defects that review caught. Four worth recording, because they are different *kinds* of failure.

**1. A login flow that could never work.** The generated `GetUser` looked correct: hash the submitted password, then `SELECT ... WHERE email = ? AND password_hash = ?`. It compiles, reads sensibly, and is fundamentally broken — bcrypt salts every hash, so hashing the same password twice yields different strings and that `WHERE` can never match. Every login would have returned 404, forever. The fix is to look up by email alone and verify in Go with `bcrypt.CompareHashAndPassword`, which re-derives using the salt embedded in the stored hash. *Caught by reasoning about bcrypt's properties, then confirmed by hashing the same password three times and observing three different outputs.*

**2. A silent deadlock.** In `MarkRepair`, the status `SELECT` ran on the transaction but the `UPDATE` ran on `s.client` — a second connection. With the pool capped at one connection, that second query waits for a connection the transaction is holding. Every call hung for the full 15-second timeout and returned 503. The same class of bug appeared earlier as a missing `defer tx.Rollback()`, which left an abandoned transaction holding the only connection and deadlocked the *entire application* permanently. *Caught by measuring: the endpoint returned in 15.002s, which is the timeout, not a slow query.*

**3. A guard that always fired.** A generated `Return` ran `UPDATE ... RETURNING`, then checked the returned row's `returned_at` to detect a double return. `RETURNING` yields **post-update** values, so `returned_at` was always non-null and the guard rejected every request. Returning a device was impossible. *Caught by running the statement directly in `sqlite3` and reading the output.*

**4. A security check that never matched.** Deleting a profile with rental history hit an `ON DELETE RESTRICT` foreign key, which was reported as a 500. The obvious fix — match `SQLITE_CONSTRAINT_FOREIGNKEY` — would have compiled, looked right, and never fired: SQLite implements `RESTRICT` through its trigger machinery, so the driver returns extended code **1811** (`SQLITE_CONSTRAINT_TRIGGER`), not **787**. *Caught by printing the actual code rather than trusting the constant's name.*

The pattern across all four: **plausible-looking code that fails only at runtime, under a condition the happy path never exercises.** Reading the diff was not enough for any of them. Each was found by executing the specific case and checking the observable behaviour — the timing, the returned row, the numeric error code.

## Deployment

Target: **GCP Compute Engine**.

SQLite needs one machine, one writer and a real filesystem. Cloud Run's disk is ephemeral and its instances multiply, so a persistent-disk VM is the honest fit. GCS FUSE and Filestore were rejected: neither provides the POSIX locking SQLite requires, and both risk a corrupted database file.

```bash
# on the VM
gcloud compute instances create hardware-hub \
  --machine-type=e2-micro --zone=<ZONE> \
  --image-family=debian-12 --image-project=debian-cloud

# database on the persistent disk
sudo mkdir -p /var/lib/hardware-hub

docker run -d --restart=unless-stopped \
  -p 80:8080 \
  -v /var/lib/hardware-hub:/app/data \
  --env-file /etc/hardware-hub.env \
  <IMAGE_REF>
```

**Deploying a new version does not touch the database.** The image holds only the binary; the database lives on the mounted volume. On boot the binary applies any pending migrations (idempotent — a restart with nothing pending is a no-op) and ensures the admin account exists without overwriting it. Roll back by running the previous image against the same volume.

Back up before a migration that is not purely additive:

```bash
sqlite3 /var/lib/hardware-hub/hardware-hub.db ".backup /var/lib/hardware-hub/pre-deploy.db"
```

Production checklist:

- `APP_ENV=production` — hides the `debug` field from error responses.
- `CORS_ORIGINS=<FRONTEND_ORIGIN>` — not `*`.
- `JWT_SECRET` from Secret Manager, never the image. Generate with `openssl rand -base64 48`.
- Rotate `ADMIN_PASSWORD` after first login. The bootstrap never overwrites an existing account, so the env value stops being authoritative the moment the admin changes it.
- TLS terminates at the load balancer or a reverse proxy; the container serves plain HTTP.
