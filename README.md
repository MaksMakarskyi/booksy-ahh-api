# Hardware Hub — API

Internal tool for Booksy employees to manage, rent and maintain company equipment. This is the **backend**: Go, SQLite, and semantic search over the inventory.

- **Live demo:** [booksy-ahh.vercel.app](https://booksy-ahh.vercel.app/) 
- **API:** [booksy-ahh-api.fly.dev](https://booksy-ahh-api.fly.dev/)
- **Frontend repository:** [https://github.com/MaksMakarskyi/booksy-ahh-website](https://github.com/MaksMakarskyi/booksy-ahh-website)

| | |
| --- | --- |
| **Read this first** | [Stack](#stack-and-why) · [Run it](#quick-start) · [API](#api-reference) · [Semantic search](#semantic-search-the-ai-layer) |
| **The graded parts** | [Status & trade-offs](#implementation-status-and-trade-offs) · [AI development log](#the-ai-development-log) · [PROMPTS.md](PROMPTS.md) |

## Stack and why

| Choice | Reason |
| --- | --- |
| **Go 1.26** | The task allows any language I'm more productive in. A single static binary makes deployment and review trivial, the compiler catches a class of mistakes before they run, and the performance and image-size story is better than Python's for the long run. My own preference, chosen deliberately over the suggested stack. |
| **SQLite** (`modernc.org/sqlite`) | Suggested by the task, and it makes review and testing frictionless — no database to install, a fresh one per test. Pure-Go driver, so `CGO_ENABLED=0` yields a static binary. For real production I would pick Postgres: SQLite has no `ENUM`, no date type, and foreign keys off by default, which pushes correctness up into application code. |
| **goose** | Versioned SQL migrations, embedded in the binary so the app self-migrates on boot. |
| **scany** | Maps rows onto structs without an ORM. Queries stay hand-written and visible. |
| **go-playground/validator** | Declarative request validation, extended with four custom rules. |
| **bcrypt** + **golang-jwt/v5** | Password hashing and HS256 access tokens. |
| **OpenAI `text-embedding-3-small`** | Vectors for semantic search. SQLite has no pgvector equivalent that works here — `sqlite-vec` is a C extension and needs CGO, which would cost the static binary. At this inventory size an index is pointless anyway: brute-force cosine over 11 vectors takes ~25µs. |
| **Fly.io** | SQLite needs one machine, one writer, a real filesystem — so a persistent volume, which Fly gives without a VM to maintain. It also terminates TLS automatically, required because an HTTPS frontend cannot call an HTTP API. Compute Engine would work but needs a reverse proxy and a domain; Cloud Run was rejected outright — its disk is ephemeral. |

## Quick start

```bash
cp .env.example .env     # fill in ADMINS, JWT_SECRET, OPENAI_API_KEY
make compose             # or: make run
```

API on `http://localhost:8080`. The container is self-sufficient — migrations are embedded and applied at boot, admins are created from `ADMINS`, and the inventory is embedded for search. No manual migration or seeding step.

```bash
TOKEN=$(curl -s -X POST localhost:8080/auth/token -H 'Content-Type: application/json' \
  -d '{"email":"you@booksy.com","password":"..."}' | jq -r .data.access_token)

curl -s localhost:8080/hardware -H "Authorization: Bearer $TOKEN" | jq
curl -s -G localhost:8080/hardware/search --data-urlencode 'query=something to test a mobile app on' \
  -H "Authorization: Bearer $TOKEN" | jq '.data[].name'
```

**Make targets:** `run` · `test` · `compose` · `format` · `tidy` · `migrate` / `migrate-down` / `migrate-status` / `migrate-reset` (development only — the binary migrates itself).

## Configuration

All configuration is environment variables; `.env` is loaded by `make` and Docker Compose. See `.env.example`.

| Variable | Required | Default | Notes |
| --- | --- | --- | --- |
| `DATABASE_URL` | **yes** | — | SQLite DSN. The pragmas are load-bearing, see below. |
| `JWT_SECRET` | **yes** | — | HS256 key. Rejected at startup under 32 bytes. |
| `ADMINS` | **yes** | — | JSON array of `{email, full_name, password}`. Created at boot. |
| `OPENAI_API_KEY` | **yes** | — | Embeddings. Search degrades without it; the app still boots. |
| `APP_ENV` | no | `production` | Controls CORS and whether errors carry a `debug` field. |
| `CORS_ORIGINS` | no | `*` | Set to the frontend origin in production. |
| `RATE_LIMIT_RPS` | no | `15` | Per IP, all routes. |
| `JWT_TTL` | no | `12h` | Access-token lifetime. |
| `SEARCH_TOP_K` | no | `5` | Results returned by `/hardware/search`. |
| `OPENAI_EMBEDDINGS_MODEL` / `EMBEDDINGS_MODEL_DIM` | no | `text-embedding-3-small` / `512` | |
| `PORT`, `GOOSE_*` | no | | `GOOSE_DRIVER`/`DBSTRING`/`MIGRATION_DIR` are only for the goose **CLI**. |

**The DSN pragmas are not optional:**

```
file:data/hardware-hub.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)
```

SQLite disables foreign keys **per connection by default** — without `foreign_keys(1)` every `REFERENCES` clause in the schema is decorative. WAL keeps readers from blocking the writer; `busy_timeout` waits for a lock instead of failing. The pool is capped at **one connection**: SQLite permits a single writer, and serialising access removes a whole class of `SQLITE_BUSY` errors at a throughput cost this app will never notice.

**Admins** are validated exactly like `POST /profiles` (company domain, password policy), the whole list before anything is written, so a typo in the third entry can't half-finish a boot. Creating an admin is idempotent and never overwrites an existing password. Treat `ADMINS` as a secret — `fly secrets set`, never `fly.toml`.

## API reference

Envelope: `{"data": ...}` on success, `{"error": {...}}` on failure. `Authorization: Bearer <token>` on everything except `/healthz` and `/auth/token`.

| Method | Path | Auth | |
| --- | --- | --- | --- |
| `GET` | `/healthz` | — | Liveness |
| `POST` | `/auth/token` | — | Email + password → access token |
| `GET` | `/hardware` | user | List all equipment |
| `GET` | `/hardware/search?query=…` | user | **Semantic search**, top k by meaning |
| `POST` | `/hardware` | **admin** | Add equipment |
| `PATCH` | `/hardware/:id` | **admin** | Edit name / brand / description / purchase date. `""` clears a description or date |
| `DELETE` | `/hardware/:id` | **admin** | Remove equipment |
| `PATCH` | `/hardware/:id/repair` · `/available` | **admin** | State transitions |
| `GET` `POST` | `/rentals` | user | List / create **your own** rentals |
| `PATCH` | `/rentals/:id/return` | user | Return **your own** rental |
| `GET` `POST` | `/profiles` | **admin** | List all accounts / create one |
| `DELETE` | `/profiles/:id` | **admin** | Remove an **employee** account |

| Status | When |
| --- | --- |
| `400` | Malformed body, unknown JSON field, failed validation, bad path parameter |
| `401` / `403` | Bad credentials or token / authenticated but not permitted |
| `404` / `409` | Missing resource / impossible transition (rent a device in repair, double return, duplicate email) |
| `429` / `503` | Rate limit / 15s request timeout |
| `500` | Our bug. Generic message to the client, full chain in the logs. |

Validation failures list every bad field at once, keyed by JSON field name, with a `request_id` echoed in `X-Request-Id` and every log line for that request.

### Business rules

- A device is rentable only when `available`. The guard is a compare-and-swap (`UPDATE … WHERE id = ? AND status = 'available'`), so two simultaneous renters cannot both win, and a partial unique index enforces one open rental per device at the storage layer regardless of application bugs.
- Renting flips the status **and** opens the rental row in one transaction — they can never disagree.
- A rented device can't go to repair; it must come back first.
- Users list and return only their own rentals, **regardless of role**.
- No self-registration — only admins create accounts, and only `@booksy.com` addresses, lowercased before validation.
- Admins see all accounts; only **employee** accounts can be deleted, and nobody can delete themselves.
- `description` and `purchase_date` are clearable (`""`); `name` and `brand` are not.
- The seed is **inventory only** — no accounts, no rentals, nothing in a state a real user can't act on.

## Semantic search (the AI layer)

`GET /hardware/search?query=something to test a mobile app on` returns the iPhone and the Galaxy, not a keyword match.

**How it works.** Each device's `name + brand + description` is embedded once and stored as a `float32` blob in `hardware_embeddings`, alongside the model that produced it and a SHA-256 of the source text. A query is embedded with the same model at request time; the API loads all vectors, ranks by cosine similarity, and returns the top 5 devices — vectors never leave the process.

**Design decisions worth knowing:**

- **Vectors live in their own table.** `hardware` has an `updated_at` trigger that fires on any update, so an embedding column would mark a device as modified when only a derived artifact changed. Separating them also makes swapping models a `DELETE`, not a migration.
- **Embedding never happens inside a transaction.** The pool holds one connection; an HTTP call mid-transaction would block every other query for a network round trip. Devices are committed first, then embedded.
- **The source hash decides staleness.** A `PATCH` that only touches `purchase_date` costs no API call. A startup backfill re-embeds anything missing or stale, and it is **non-fatal** — an OpenAI outage degrades search rather than stopping the service from booting.
- **Data-quality notes are excluded from the embedded text.** They were up to 28% of some devices' vectors and measurably distorted ranking; they stay in the column for users and for the audit trail.
- **Vectors are normalized**, so cosine is a dot product; similarity is computed once per candidate rather than inside the comparator.
- **No vector index, deliberately.** `sqlite-vec` is a loadable C extension and needs CGO, costing the static binary. Brute force over 11 vectors is ~25µs; an index earns its keep somewhere past 100k rows.

## Architecture

```
cmd/api/         config → dependencies → migrate → admins → embeddings → serve
internal/
  auth/          login, JWT middleware, current-user context
  hardware/      CRUD, repair transitions, semantic search, embedding backfill
  rentals/       rent / return / list
  profiles/      accounts + startup admin bootstrap   (roles/ is a leaf package)
  server/        router, CORS, config/, dependencies/
  utils/         errors, validation, jwt, password, embeddings, migrate
migrations/      embedded SQL
```

Every domain package has the same shape: `build.go` (composition root), `handler.go`, `store.go`, `models.go`, and `requests.go` where the DTOs earn a file.

- **Layering.** `utils/` and `roles/` import nothing internal. Domain packages import `dependencies`, so anything the registry holds must sit *below* them — that's why `jwt` deals in plain strings, why `roles` is its own package, and why `Embedder` lives in `utils/embeddings`. Three import cycles were resolved this way.
- **Store interfaces** are declared next to the handler that consumes them, with `SQLiteStore` implementing them — swapping databases means one adapter, and tests can substitute a fake.
- **Handlers never choose status codes.** They wrap errors with context and return them; one `HTTPErrorHandler` maps sentinel errors onto codes, logs the full chain, and returns a client-safe message.
- **Strict JSON.** `DisallowUnknownFields`, so `{"stauts": …}` is a 400 rather than a silent no-op, and a field the client doesn't own (`status`) can't be smuggled in.
- **Validation.** Four custom rules: `date`, `notfuture`, `password`, `maxbytes`. Payload types opt into `Normalizer` (trim/lowercase before validation) and `SelfValidator` (cross-field rules).

## Data model

```
profiles                hardware                      rentals                 hardware_embeddings
--------                --------                      -------                 -------------------
id                      id                            id                      hardware_id  → hardware (CASCADE)
email         UNIQUE    name, brand                   hardware_id →CASCADE    model, dimensions
full_name               description    NOT NULL ''    user_id     →CASCADE    source_hash
role          CHECK     purchase_date  nullable       rented_at               vector  BLOB
password_hash           status         CHECK          returned_at             CHECK len = dims*4
```

- `status` ∈ `available | in_use | repair` and `role` ∈ `employee | admin`, both `CHECK` constraints — SQLite has no `ENUM`.
- `rentals` is append-only history; a return sets `returned_at`. `hardware.status` is a fast projection of it.
- **`description` is `NOT NULL DEFAULT ''` but `purchase_date` is nullable, on purpose.** For free text, "no description" and "empty description" are the same fact. A date is different: `''` is not a date but a sentinel, and SQL compares it — `WHERE purchase_date < '2023-01-01'` would match it, and `MIN()` would return it. `NULL` is what SQL provides for unknown.
- Timestamps are TEXT RFC 3339 UTC, dates TEXT `YYYY-MM-DD` — ISO-8601 sorts correctly as a string. `updated_at` is maintained by triggers (no `ON UPDATE CURRENT_TIMESTAMP` in SQLite).

## Implementation status and trade-offs

### ✅ Fully implemented

- **Management:** admin adds / edits / deletes equipment, toggles repair, creates and removes accounts. Login issues HS256 JWTs; no self-registration.
- **Rentals:** rent and return with guards against every impossible state, concurrency-safe via compare-and-swap plus a partial unique index.
- **AI layer:** semantic search over the inventory, with write-through embedding and an idempotent startup backfill.
- **Cross-cutting:** role-based authorisation from a signed claim re-validated per request; centralised error handling; structured logging with request-id correlation; strict JSON; per-field validation errors; per-IP rate limiting; self-migrating binary.

### ⚡ Shortcuts and "hacks"

**1. Filtering and sorting happen on the client.** Rows are ~1KB, so a thousand devices is a single small fetch; client-side filtering is instant and keeps dynamic `WHERE`/`ORDER BY` assembly (and its injection surface) out of the backend. *This holds until pagination is needed* — filtering and pagination must live on the same side, or you filter only the loaded page. *Fix:* `GetAll(ctx)` → `List(ctx, filters)`, confined to one store and one handler.

**2. 12-hour access tokens, no refresh flow.** Acceptable for an internal tool with a single-day session. *Fix:* 15-minute access token plus a rotating refresh token in an httpOnly cookie.

**3. The token lives in `localStorage`.** Frontend and API are on different origins, so a cookie needs `SameSite=None; Secure` plus credentialed CORS. **Stated plainly: `localStorage` is readable by any injected script, so an XSS becomes token theft.** Exposure is bounded by the token lifetime. *Fix:* as above.

**4. Embedding runs inside the request.** A create or update waits on OpenAI (~100–400ms) before responding. It's synchronous so the device is searchable the moment the response returns, and failures are logged rather than returned — an outage must not stop an admin adding hardware. *Fix:* a real queue and worker (roadmap 8).

### ⚠ Partial / missing

- **No pagination.** Deliberate, per shortcut 1.
- **"Unknown" has two spellings in `purchase_date`.** The seed writes `NULL`, the API writes `''`. Reads paper over it (both falsy in the client), but a server-side `WHERE purchase_date IS NULL` would miss half the rows.
- **Deleting an admin answers `404`, not `403`.** The delete matches `role = 'employee'`, and a statement matching nothing is indistinguishable from a missing row. A status-code wart, not a security hole.
- **No relevance floor on search.** Top-K always returns K results, so with 11 devices the tail can be irrelevant. A minimum cosine threshold is the fix.

### 🔮 Next steps — the 24-hour roadmap

1. **Store-level tests for failure paths.** The end-to-end suite can't easily provoke a database error, so `ErrStoreInternal` branches are the main uncovered code. An injected failing driver closes the gap.
2. **A relevance threshold on search**, so a vague query returns three good results instead of five with two duds.
3. **Refresh tokens and finer rate limiting**, keyed on the JWT `sub` rather than IP for authenticated routes.
4. **Soft-delete for accounts with rental history** — blank the personal data and set a `deleted` flag rather than cascading the whole history for the user.
5. **One spelling for an unknown `purchase_date`.** Mapping `''` → `NULL` means replacing `COALESCE($4, …)` with a `CASE`, since `COALESCE` cannot set a column to `NULL`. An `Optional[T]` wrapper would make it a type rather than a convention.
6. **Clarify admin boundaries** — can an admin manage employees' rentals? Should renting need approval?
7. **UX**: notify users when a device returns to `available`, and a queue for contested devices instead of polling the list.
8. **A real embedding queue and worker.** Today embedding happens in the handler, so the client waits for something they didn't ask for. A queue plus a small Cloud Function would make it a true background job, with a dead-letter queue so a failed embedding can be replayed after the cause is fixed rather than just logged.
9. **AI-generated descriptions.** Let admins opt into generating a device description on create or update — good descriptions are what make search work, and writing them by hand is the slow part.
10. **Move to Postgres** and decouple the store from the app. The current single-container coupling already rules out platforms like Cloud Run; the earlier the migration, the cheaper it is.
11. **Stop hand-rolling auth.** Supabase Auth or similar brings key rotation, TTL management and a client library that are tested and working.

## Testing

```bash
make test
```

**126 table-driven end-to-end cases, 73.1% statement coverage** of `internal/…`. They drive the real router, middleware and SQLite store — only the database is disposable, one per test in `t.TempDir()`, and each test runs the real migrations and startup bootstrap, so a broken migration fails the build.

That's deliberate over unit tests with mocked stores: nearly every defect this project hit lived *between* layers — a query outside its transaction, a sentinel that never reached the error handler, a `RETURNING` clause whose post-update semantics inverted a guard, a sort whose direction flipped during a refactor. A mocked store sees none of those.

Request bodies are raw JSON strings, so each case shows exactly what goes on the wire — including payloads no Go struct could express, like a misspelled field. Fixtures go through the public API, never `INSERT`, so a test can only set up state a real operator could. The embedder is swapped for a deterministic fake, so the suite needs no API key, no network and no budget. The uncovered remainder is mostly `ErrStoreInternal` branches, which need a database-level failure to reach.

## The AI development log

### Tooling

| Tool | Used for |
| --- | --- |
| **Claude Code** | The main tool. Brainstorming architecture and having my proposals challenged, completing parts of the codebase (tests, helpers, fixes), and fast research on library APIs and behaviour. |
| **CodeRabbit** | Ran after most commits to flag bugs and typos. Commits named "Added improvements after code review" are usually applying its findings. |
| **GitHub Copilot** | Small functionality and mechanical edits like renaming across several places at once. |

### Data strategy — auditing the seed

The provided dataset is deliberately dirty. Every record was audited before insertion and each deviation is a numbered decision (`[D-1]`…`[D-9]`) in `migrations/002_seed.sql`, with the reasoning in the file itself.

| Issue in source | Decision |
| --- | --- |
| Duplicate `id: 4` | A primary key can't hold both. First occurrence keeps the id; the second is appended at 12 rather than backfilling the unexplained gap at 8. |
| Brand `"Appel"`, date `"22-05-2023"` | Corrected — an unambiguous misspelling, and a date that must be in the format YYYY-MM-DD instead of DD-MM-YYYY. |
| Date `"2027-10-10"` (future) | **Nulled, not guessed.** Any plausible correction is invention, and invented data is worse than absent data. Raw value preserved in the description. |
| Status `"Unknown"` | Not a value the model has. Quarantined as `repair` so it can't be rented until a human identifies it. A fourth status is the better long-term model — a deliberate deferral. |
| Dell XPS: `"Available"` **with** *"Battery swelling, do not issue without service."* | **The note wins.** Forced to `repair`. Trusting the status field would let the app hand someone a fire hazard. The one place the migration deliberately contradicts its source. |
| MacBook Air: `"Available"` with liquid damage in `history` | Same reasoning — forced to `repair`. |
| `"In Use"` devices (one with `assignedTo`, one without) | Seeded as `available`, assignment recorded in the description. `in_use` needs an open rental, and a rental needs an account that can return it — placeholder accounts nobody can log into would freeze those devices forever. |
| Empty brand on the unknown device | Replaced with `"Unknown Brand"`. `brand` is `NOT NULL` with `min=1` in the API, so a blank brand is a row the application itself could never create. |
| Missing descriptions (8 of 11) | **Authored, not sourced** — see `[D-9]`. Semantic search embeds `name + brand + description`, and "Apple iPhone 13 Pro Max / Apple" is too little text to sit near "something to test a mobile app on". Each is catalogue copy about a well-known product, clearly the operator's own text, asserting nothing about the source record. |

The principle: **normalise what is unambiguous, quarantine what is not, and never invent** — where "invent" means fabricating a missing source *value*, not writing your own product copy.

### Prompt trail

**[PROMPTS.md](PROMPTS.md)** — the session's prompts, with the turns that shaped the architecture quoted and annotated with what each settled.

### The "correction" — three times the AI was wrong

**1. A login flow that could never work.** The generated `GetUser` hashed the submitted password, then `SELECT … WHERE email = ? AND password_hash = ?`. It compiles and reads sensibly, and is fundamentally broken: bcrypt salts every hash, so hashing the same password twice yields different strings and that `WHERE` can never match. Every login would have returned 404, forever. The fix is to look up by email alone and verify in Go with `bcrypt.CompareHashAndPassword`. *Caught by reasoning about bcrypt's properties, then confirmed by hashing one password three times and getting three different outputs.*

**2. Seed data that locked itself.** The AI proposed keeping the source's `"in_use"` devices and inventing placeholder accounts with garbage emails and `!` passwords, plus rental rows binding them — purely to satisfy the invariant that `in_use` needs a rental. I rejected it: only the renting account can end a rental, and nobody could log in as those placeholders, so two devices would have been unavailable forever. The seed now carries inventory only. *Caught by asking what the seed would mean in production rather than whether it satisfied the schema.*

**3. A silent deadlock.** In `MarkRepair` the status `SELECT` ran on the transaction but the `UPDATE` ran on `s.client` — a second connection. With the pool capped at one, that query waited for a connection the transaction was holding. Every call hung for the full 15-second timeout and returned 503. *Caught by measuring: the endpoint returned in 15.002s, which is a timeout, not a slow query.* The same class of bug shaped the embedding design — an HTTP call inside a transaction would hold that single connection for a whole network round trip, which is why embedding happens after the commit.

## Deployment

Deployed to Fly.io — see the stack table for why. `fly.toml` is committed; secrets are not.

```bash
fly apps create <APP_NAME>
fly volumes create hardware_hub_data --region fra --size 1 --yes
fly secrets set JWT_SECRET="$(openssl rand -base64 48)" OPENAI_API_KEY='sk-...'
fly secrets set ADMINS='[{"email":"you@booksy.com","full_name":"You","password":"..."}]'
fly deploy --ha=false
```

**`--ha=false` is not optional.** Fly creates two machines by default; with SQLite that's two independent databases behind one hostname, and requests would see different data depending on which answered. One volume, one machine.

The image holds only the binary — the database lives on the volume, so a release never touches it. On boot the binary applies pending migrations, ensures the admins exist, and backfills any missing embeddings, all idempotent.

**Migrations are versioned, so editing an applied one does nothing.** A seed change needs the database discarded:

```bash
fly ssh console -C "rm -f /app/data/hardware-hub.db /app/data/hardware-hub.db-wal /app/data/hardware-hub.db-shm"
fly deploy --ha=false
```

**Production checklist:** 
- `APP_ENV=production` · `CORS_ORIGINS=<FRONTEND_ORIGIN>` not `*` · `JWT_SECRET`, `ADMINS` and `OPENAI_API_KEY` via `fly secrets` 
- rotate the bootstrap passwords after first login (the values stay in your secrets store) 
- back up before a non-additive migration with `fly ssh console -C "cp /app/data/hardware-hub.db /app/data/pre-deploy.db"`.
