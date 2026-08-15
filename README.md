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
- [Testing](#testing)
- [The AI development log](#the-ai-development-log)
- [Deployment](#deployment)

## Stack and why

| Choice | Reason |
| --- | --- |
| **Go 1.26** | The task allows any language in which you are more productive. Go gives a single static binary, which makes deployment and review trivial. Moreover, Go is a compiled language, which gives some extra type safety for the application. Also, the language allows you to implement advanced concurrency patterns using goroutines and channels, which are features no other language gives us. Go brings other advantages as well, including smaller image size and better performance according to numerous articles and case studies written by employees from Big Tech companies that you can find on the internet. So, from my personal point of view, Go is a better choice for the long run to some extent than something like Python. Therefore, I decided to pick Go for this backend. |
| **SQLite** (`modernc.org/sqlite`) | Suggested by the task. Pure-Go driver, so `CGO_ENABLED=0` produces a static binary with no libc dependency. SQLite was chosen mainly because it allows a seamless testing experience, though I would definitely pick something like Postgres for a real production app, since SQLite is quite limited. For example, it does not allow creating enumerators and stores dates as strings, which removes some database-level checks and may result in invalid DB states if a developer is not careful and does not ensure those checks at the application level. |
| **goose** | Versioned SQL migrations, embedded into the binary so the app self-migrates on boot. |
| **scany** | Maps SQL rows onto structs without an ORM. Queries stay hand-written and visible. |
| **go-playground/validator** | Declarative request validation, extended with three custom rules. |
| **bcrypt** + **golang-jwt/v5** | Password hashing and HS256 access tokens. |
| **Fly.io** (deployment) | A container platform with first-class persistent volumes, which is what SQLite needs: one machine, one writer, a real filesystem. It also terminates TLS on `*.fly.dev` automatically — required, because the frontend is served over HTTPS and a browser will not let an HTTPS page call an HTTP API. GCP Compute Engine was the alternative and would work equally well, but it needs a reverse proxy and a domain to obtain a certificate; Cloud Run was rejected because its disk is ephemeral. |

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
| `RATE_LIMIT_RPS` | no | `15` | Requests per second per IP, across all routes. |
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
| `GET` | `/profiles` | **admin** | List every account, employees and admins alike |
| `POST` | `/profiles` | **admin** | Create an account |
| `DELETE` | `/profiles/:id` | **admin** | Remove an **employee** account |

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
| `429` | Rate limit exceeded — see `RATE_LIMIT_RPS` |
| `503` | Request exceeded the 15s timeout |
| `500` | Our bug. The client message is always generic; the full chain is logged. |

### Business rules

- A device can only be rented when `available`. The guard is a compare-and-swap (`UPDATE ... WHERE id = ? AND status = 'available'`), so two simultaneous renters cannot both win.
- Renting flips the status **and** opens a rental row **in one transaction** — the two can never disagree.
- A partial unique index (`rentals_one_active_per_hardware_idx`) enforces at most one open rental per device at the storage layer, independent of application code.
- A rented device cannot be sent to repair; it must be returned first, otherwise the open rental would be orphaned.
- Users list and return only their own rentals, **regardless of role**.
- Accounts cannot self-register. Only an admin creates accounts.
- Admins see the full account list, their colleagues included. Deletion is narrower than listing: the `DELETE` statement matches `role = 'employee'`, so an admin account can be listed but never removed through the API, and no one can delete themselves. Both refusals are deliberate — demoting or removing the last admin would lock everybody out of account management.
- Only `@booksy.com` addresses may hold an account. Enforced on account creation and on the startup admin bootstrap, after the address is lowercased, so `Name@BOOKSY.COM` is accepted and stored as `name@booksy.com`.
- The seed contains **inventory only** — no accounts and no rentals. The only account on a fresh install is the admin created from `ADMIN_EMAIL` / `ADMIN_PASSWORD`.

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

Every domain package has the same shape: `build.go` (composition root), `handler.go` (HTTP), `store.go` (SQL), `models.go` and, where the request DTOs are numerous enough to earn their own file, `requests.go` — `hardware` keeps its two payload types in `models.go` instead.

**Layering.** `utils/` and `roles/` are leaves that import nothing internal. Domain packages import `dependencies`, so anything the registry holds must sit *below* the domain — this is why the `jwt` package deals in plain strings rather than domain types, and why `roles` is its own package. Two import cycles were resolved this way during development.

**The store interface.** Each domain declares a `Store` interface next to the handler that consumes it, with `SQLiteStore` implementing it. Handlers depend on the interface, not the implementation, so swapping the database means writing one adapter — and tests can substitute an in-memory fake with no database at all.

**Error handling.** Handlers never choose status codes. They wrap errors with context (`fmt.Errorf("...: %w", err)`) and return them; a single `HTTPErrorHandler` maps sentinel errors (`ErrStoreNotFound`, `ErrStoreConflict`, `ErrStoreForbidden`, ...) onto status codes, logs the full chain, and returns a client-safe message. 5xx responses never leak internals; the detail is in the logs, correlated by request id.

**Validation.** Request structs carry `validate` tags. Three custom rules are registered: `notfuture` (a date not in the future), `password` (character-class policy) and `maxbytes` (a length limit in bytes rather than runes, so a multi-byte name cannot overflow a column). The company-domain rule needs no custom code — it is the built-in `endswith`. Payload types can opt into two interfaces — `Normalizer` (trim/lowercase before validation) and `SelfValidator` (rules spanning several fields, such as "a PATCH must change something").

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
- Centralised error handling, structured logging with request-id correlation, strict JSON decoding, per-field validation errors, and a per-IP rate limit on every route.
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

- **AI layer.** Not implemented. Semantic search is the intended feature; the store interface is the seam it would attach to.
- **PATCH cannot clear a nullable field.** `*string` cannot distinguish "absent" from "explicit null" — both decode to `nil`, and `COALESCE` reads `nil` as "keep". So `description` can be set and changed, never removed. Fixing it needs an `Optional[T]` wrapper that records key presence.
- **No pagination.** Deliberate, per shortcut 1.
- **Deleting an admin answers `404`, not `403`.** `GET /profiles` returns every account, so the client can see admin rows it is not allowed to remove. The delete query matches `role = 'employee'`, and a statement that matches nothing is indistinguishable from a missing row, so the honest "you may not delete an admin" is reported as "no such profile". The fix is to look the row up before deleting and return a dedicated error; it is a status-code wart, not a security hole.
- **Frontend.** Lives in a separate repository *(link to be added)*.

### 🔮 Next steps — the 24-hour roadmap

1. **Store-level tests for the failure paths.** The end-to-end suite cannot easily provoke a database error, so the `ErrStoreInternal` branches are the main uncovered code. An injected failing driver would close that gap.
2. **The AI layer — semantic search.** Embed each device's name, brand and description at write time, store the vectors alongside the rows, and rank by cosine similarity at query time. `"something to test a mobile app on"` should return the iPhone and the Galaxy.
3. **Refresh tokens and rate limiting.** Shorten the access token, add rotation, and put a more granular rate limiter on the endpoints, possibly selecting more thoughtful limiting keys like the JWT("sub") key instead of the IP address for protected routes.
4. **Come up with a way to delete a user with rental history.** For example, make a delete operation work in the way that the account is being blocked forever from access to it (set the password_hash equal to "!") and delete user data such as `"email"` and `"full_name"` or set them to some random string and "Deleted Employee" values. Or just add a flag called deleted that allows distinguishing between the deleted account and active ones at the DB and Application levels.
5. **Improve JSON decode and PATCH updates.** Introduce `Optional[T]` wrapper for the fields that might be set to an empty string instead of `*string`.
6. **Improve business logic.** Clarify the boundaries of admin capabilities. For example, can an admin manage the rentals of the employees? Or, does the employee need to request approval of the hardware rental?
7. **Add tiny functionalities for better UX.** For example, add notifications for the users that will notify them when the needed hardware is back to `"available"` status. Or, add a queue for the hardware, so that employees just put them into the virtual queue to get the awaited hardware instead of constantly monitoring for the device availability.
8. **Move to a more production-ready store.** Migrate to a more mature database such as Postgres to have broader functionality in the future. The earlier the migration happens, the easier it will be to do it. Additionally, using something like Supabase would allow you to make changes to the user interface, allowing you to fix things and control the most important things, such as admin management, manually.
9. **Decouple the API application and the store.** Now, the application and the store are titly couples and deployed with a single Docker container. For now, it works well, but it will create complexities in the future. For example, such a setup already limits the deployment options, and great services such as GCP Cloud Run cannot be selected by design, which has an effect on the maintainability and infrastructure costs.
10. **Stop implementing the auth on your own.** It is better and more secure to use services such as Supabase Auth to have a more secure system overall and features such as a client library, key rotation, and TTL management out of the box, tested and working.

## Testing

```bash
make test
```

Table-driven end-to-end tests covering every handler. They drive the **real** router, middleware stack and SQLite store — only the database file is disposable, created per test in `t.TempDir()`.

That is a deliberate choice over unit tests with mocked stores. Nearly every defect this project hit during development lived in the seam *between* layers: a query issued outside its transaction, a sentinel error that never reached the error handler, middleware that dropped the user from the request context, a `RETURNING` clause whose post-update semantics inverted a guard. A mocked store sees none of those.

| File | Covers |
| --- | --- |
| `tests/api_test.go` | the harness — the whole vocabulary the other four files use |
| `tests/auth_test.go` | login, indistinguishable failure modes, malformed payloads, token rejection, rate limiting |
| `tests/hardware_test.go` | CRUD, validation, repair transitions, admin-only writes, strict JSON |
| `tests/rentals_test.go` | rent/return round trip, double-rent, ownership scoping, conflicts |
| `tests/profiles_test.go` | account creation, the `@booksy.com` rule, password policy, delete guards |

The harness is eight names, and there is nothing else to learn before reading a test:

| Helper | Returns |
| --- | --- |
| `newAPI(t)` | a running app on an empty database, with a logged-in admin in `a.admin` |
| `a.call(token, method, path, body)` | one request → `(status, body)` |
| `a.login(email, password)` | a bearer token |
| `a.employee(email)` | a bearer token and a profile id |
| `a.device(name)` | a hardware id |
| `a.rent(token, deviceID)` | a rental id |
| `field(t, body, "data.status")` | one value from a JSON body, by dotted path |
| `count(t, body, "data")` | the length of a JSON array |

Request bodies in the tables are raw JSON strings rather than Go values, so each case shows exactly what goes on the wire — including payloads no Go struct could express, such as a misspelled field or a field the DTO deliberately does not have. Fixtures are created through the public API, never with `INSERT`, so a test can only set up state a real operator could.

The rate limit is configurable (`RATE_LIMIT_RPS`) largely because of this suite: the production value of 15 requests per second throttles a test that drives the API as fast as the CPU allows. The harness raises it, and one test lowers it again to prove the middleware is still wired up.

Statement coverage of `internal/...` is **75.7%**. The uncovered remainder is mostly `ErrStoreInternal` branches, which require a database-level failure to reach.

The suite also exercises startup: every test runs the real migrations and the real admin bootstrap, so a broken migration or a bootstrap regression fails the build.

## The AI development log

### Tooling

| Tool | Used for |
| --- | --- |
| Claude Code | Claude Code was the main tool in the AI stack. Used for brainstorming the architecture and questioning the proposed solutions to receive feedback and a list of areas for improvement. Claude Code helped to complete several parts of the codebase, including some tests, helper functions, and minor fixes to the logic. Moreover, the tool allowed for better and faster information collection on the libraries and tooling work principles and their APIs. Finally, Claude Code allowed for automating routine work such as writing the README.md and exporting the API spec for the Agent working on the frontend. |
| CodeRabbit | CodeRabbit was used after every commit to validate the code and highlight bugs, typos, and other kinds of issues. If you see a commit with a name such as "Added improvements after code review", it is very likely that the additional commit was needed to apply some changes proposed by the CodeRabbit AI Agent. |
| GitHub Copilot | GitHub Copilot was used to implement small functionalities and speed up things like the manual rewriting of object names in a few places at the same time. |

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
| `assignedTo: "j.doe@booksy.com"` on an in-use device | Recorded in the device description and seeded as `available`. An earlier revision created a placeholder account and an open rental to keep `in_use` consistent — but placeholders have no usable password, and a rental may only be returned by its owner, so those devices were permanently stuck. Seeding inventory only keeps every row in a state a real user can act on. |
| `"In Use"` with **no** assignee | Same resolution: seeded as `available`. Claiming a device is on the shelf when someone physically holds it is a real loss of information, but it is recoverable — an admin re-issues it through the app. A device frozen in `in_use` with no one able to return it is not. |
| Empty brand, null date | Kept as-is. Absent is a legitimate value. |

The general principle: **normalise what is unambiguous, quarantine what is not, and never invent.**

### Prompt trail

**[PROMPTS.md](PROMPTS.md)** — the 41 prompts of the session, with the twelve that shaped the architecture quoted in full and annotated with what each one settled.

The turns worth reading first: the [stack change](PROMPTS.md#3-python--supabase--go--sqlite) from Python + Supabase to Go + SQLite, the [interface-first instruction](PROMPTS.md#2-interfaces-before-implementations) that made that change cheap, the [action-routes-vs-REST](PROMPTS.md#9-action-routes-vs-rest) discussion behind `PATCH /hardware/:id/repair`, and the [seed cleanup](PROMPTS.md#11-a-seed-that-nobody-could-unstick) that reversed an AI proposal.

### The "correction"

AI assistance produced several defects that the review caught. The following are worth mentioning:

**1. Seed data decisions.** The AI proposed leaving the `"in_use"` hardware with the same status but inventing accounts with garbage emails and `!` passwords, and adding rental records binding those new users and hardware pieces just to ensure that the DB has no invalid state, since every `"in_use"` hardware requires a rental row and a user who rents it by design. I decided to move away from this and seed those hardware rows with the `"available"` status. Otherwise, those devices would be locked forever, because only the renting account can return the hardware and end the rental. Since the proposed accounts were garbage, nobody could log in and return the hardware to `"available"` status, resulting in permanent lock. Thus, the app only loads one admin account into profiles at startup if not exist, allowing the platform admin to manage the initial products and add employees. *Caught by reasoning about the ideal seed state, so that it functions in the way it would be in the case of a real production app.*

**2. A login flow that could never work.** The generated `GetUser` looked correct: hash the submitted password, then `SELECT ... WHERE email = ? AND password_hash = ?`. It compiles, reads sensibly, and is fundamentally broken — bcrypt salts every hash, so hashing the same password twice yields different strings and that `WHERE` can never match. Every login would have returned 404, forever. The fix is to look up by email alone and verify in Go with `bcrypt.CompareHashAndPassword`, which re-derives using the salt embedded in the stored hash. *Caught by reasoning about bcrypt's properties, then confirmed by hashing the same password three times and observing three different outputs.*

**3. A silent deadlock.** In `MarkRepair`, the status `SELECT` ran on the transaction but the `UPDATE` ran on `s.client` — a second connection. With the pool capped at one connection, that second query waits for a connection the transaction is holding. Every call hung for the full 15-second timeout and returned 503. The same class of bug appeared earlier as a missing `defer tx.Rollback()`, which left an abandoned transaction holding the only connection and deadlocked the *entire application* permanently. *Caught by measuring: the endpoint returned in 15.002s, which is the timeout, not a slow query.*

## Deployment

Deployed to **Fly.io**. SQLite needs one machine, one writer and a real filesystem, so the deployment target has to offer a persistent volume — and the frontend is served over HTTPS, so the API needs a certificate or the browser blocks the calls as mixed content. Fly gives both without a VM to maintain.

`fly.toml` is committed. Secrets are not.

```bash
fly apps create <APP_NAME>
fly volumes create hardware_hub_data --region fra --size 1 --yes
fly secrets set JWT_SECRET="$(openssl rand -base64 48)" ADMIN_PASSWORD='<STRONG_PASSWORD>'
fly deploy --ha=false
```

`--ha=false` is not optional. Fly creates two machines by default; with SQLite that means two independent database files behind one hostname, and requests would see different data depending on which machine answered. **One volume, one machine.**

### What survives a deploy

The image holds only the binary. The database lives on the mounted volume, so a new release never touches it. On boot the binary applies any pending migrations — idempotent, a no-op when there is nothing new — and ensures the admin account exists without overwriting an existing one.

`auto_stop_machines` lets the machine sleep when idle and wake on the next request. The volume persists across stops, so only the machine is transient. Set `min_machines_running = 1` to avoid cold starts.

### Rolling out a seed change

Migrations are versioned, so goose will not re-run `002_seed.sql` on a database that already recorded version 2. A seed edit therefore needs the database discarded, not just a redeploy:

```bash
fly ssh console -C "rm -f /app/data/hardware-hub.db /app/data/hardware-hub.db-wal /app/data/hardware-hub.db-shm"
fly apps restart <APP_NAME>
```

The next boot finds an empty volume, runs both migrations from scratch and recreates the admin.

### Production checklist

- `APP_ENV=production` — hides the `debug` field from error responses.
- `CORS_ORIGINS=<FRONTEND_ORIGIN>` — not `*`.
- `JWT_SECRET` via `fly secrets`, never in `fly.toml`. Generate with `openssl rand -base64 48`.
- Rotate `ADMIN_PASSWORD` after first login. The bootstrap never overwrites an existing account, so the env value stops being authoritative the moment the admin changes it.
- Back up before a migration that is not purely additive:
  ```bash
  fly ssh console -C "cp /app/data/hardware-hub.db /app/data/pre-deploy.db"
  ```

### The alternative: GCP Compute Engine

Equally valid and documented here because it was the original plan. An `e2-micro` with a persistent disk gives the same one-machine-one-writer guarantee, and the container mounts `/var/lib/hardware-hub` for its data. The cost is TLS: a bare VM serves plain HTTP, so it needs a reverse proxy (Caddy obtains Let's Encrypt certificates automatically) plus a hostname — a real domain, or wildcard DNS such as `sslip.io`, which resolves an IP embedded in the name and is enough for Let's Encrypt to issue.

Cloud Run was rejected. Its filesystem is ephemeral and its instances multiply, and the volume types it does support — GCS FUSE and Filestore — do not provide the POSIX locking SQLite requires. Google's own documentation warns against running a database on gcsfuse.
