# Implementation Report

Living document: updated after each completed task. Architecture and rationale live in
[design.md](design.md); this file records **what exists, what was verified, and what changed**.

Last updated: 2026-11-01 · Status: **Complete — all required features delivered**

## Progress

| # | Task | Status |
|---|------|--------|
| 1 | Skeleton, Compose, database migrations | ✅ Done |
| 2 | Backend foundation (config, DB, router, errors) | ✅ Done |
| 3 | Auth module (register / login / logout / me) | ✅ Done |
| 4 | Hardening + token edge cases | ✅ Done |
| 5 | Task module (CRUD, filter, ownership) | ✅ Done |
| 6 | Backend integration test suite | ✅ Done |
| 7 | Frontend foundation (Vite, types, API client) | ✅ Done |
| 8 | Auth UI | ✅ Done |
| 9 | Tasks UI | ✅ Done |
| 10 | Docs and final verification | ✅ Done |

## Environment

| Item | Value |
|---|---|
| Go | 1.25+ required (pgx v5.12 floor); verified with 1.26.5 |
| Node | 22 (frontend) |
| PostgreSQL | 17-alpine in Docker, host port 5432 |
| Backend deps | `gin v1.11.0`, `pgx/v5 v5.12.0`, `golang-jwt/jwt/v5 v5.3.1` |
| Migrations | `migrate/migrate:v4.19.0` (one-shot Compose service) |

### Skills installed (project-local, tracked in `skills-lock.json`)

| Skill | Purpose |
|---|---|
| `golang-testing` | table-driven tests, integration test patterns |
| `golang-security` | cookie/crypto/injection guidance, gosec |
| `golang-popular-libraries` | dependency vetting |
| `golang-gin-api` | Gin routing, binding, middleware, error mapping |
| `golang-project-layout` | `cmd/` + `internal/` layout, module naming |
| `auth-implementation-patterns` | JWT/session patterns and pitfalls |
| `postgresql-table-design` | schema types, constraints, indexes |
| `docker-compose-patterns` | healthchecks, ordering, volumes |
| `tdd` | red-green-refactor workflow |
| `design-taste-frontend`, `web-design-guidelines`, `impeccable` | UI tasks (pre-existing) |

---

## Task 1 — Skeleton, Compose, migrations ✅

### What exists

| File | Purpose |
|---|---|
| `docker-compose.yml` | `db` (Postgres 17, healthcheck, named volume) and one-shot `migrate` |
| `backend/migrations/000001_init.up.sql` | `users`, `tasks`, constraints, index |
| `backend/migrations/000001_init.down.sql` | drops tables in FK order |
| `Makefile` | `db-up`, `down`, `db-reset`, `migrate-up`, `migrate-down`, `migrate-version`, `api`, `test` |
| `.env.example` | full configuration contract with dev defaults |
| `.gitignore` | `.env`, `node_modules`, build output |
| `README.md` | stack, quick start, layout |

### Schema

```
users(id BIGINT IDENTITY PK, email TEXT NOT NULL, password_hash TEXT NOT NULL,
      created_at TIMESTAMPTZ NOT NULL DEFAULT now())
  UNIQUE INDEX ON (LOWER(email))          -- case-insensitive uniqueness in the DB
  CHECK (char_length(email) <= 254)

tasks(id BIGINT IDENTITY PK, user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      title TEXT NOT NULL CHECK (char_length(title) BETWEEN 1 AND 255),
      description TEXT NOT NULL DEFAULT '' CHECK (char_length(description) <= 2000),
      completed BOOLEAN NOT NULL DEFAULT false,
      created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
      updated_at TIMESTAMPTZ NOT NULL DEFAULT now())
  INDEX (user_id, created_at DESC, id DESC)
```

### Decisions

- **Migrations run in a separate container**, never from the API binary. The app needs no schema
  privileges, and Compose orders `migrate` before the API.
- **`BIGINT IDENTITY`** instead of UUID: IDs are not secrets, ownership is enforced in SQL.
- **`LOWER(email)` unique index** makes case-insensitive uniqueness a database guarantee.
- **CHECK constraints mirror service validation** as defense-in-depth.

### Verification

```
make db-up        → container healthy
make migrate-up   → 1/u init
\d users, \d tasks → schema as above
constraints       → duplicate OWNER@example.com rejected; empty title rejected;
                    2001-char description rejected; user delete cascaded to tasks
make migrate-down → 1/d init, only schema_migrations left
make migrate-up   → 1/u init, version = 1
```

---

## Task 2 — Backend foundation ✅

### What exists

| File | Purpose |
|---|---|
| `backend/cmd/server/main.go` | config load, logger, pgx pool, dependency wiring, HTTP server, signal handling |
| `backend/internal/config/config.go` | env parsing, defaults, aggregated fail-fast errors |
| `backend/internal/config/config_test.go` | 5 tests |
| `backend/internal/database/postgres.go` | `pgxpool` creation + startup ping |
| `backend/internal/errors/errors.go` | `AppError`, constructors |
| `backend/internal/errors/http.go` | single `Fail` writer, JSON envelope, binding errors |
| `backend/internal/logger/logger.go` | `slog` setup (JSON in production, text otherwise) |
| `backend/internal/middleware/logger.go` | structured request log via `slog` |
| `backend/internal/middleware/recovery.go` | panic → JSON 500, stack logged server-side |
| `backend/internal/handler/router.go` | router assembly, `/healthz`, 404/405 handlers |
| `backend/internal/handler/router_test.go` | health, 404, 405, panic recovery |

### Endpoints

| Method | Path | Behavior |
|---|---|---|
| GET | `/healthz` | `200 {"status":"ok"}`; `503` if the DB ping fails |
| * | anything else | `404` / `405` in the shared error envelope |

Error envelope (all failures):

```json
{"error":{"code":"validation_failed","message":"some fields are invalid","fields":{"email":"..."}}}
```

### Decisions

- **Config is loaded once and fails fast**, reporting every problem at once. Nothing else reads
  `os.Getenv`. Production refuses to start unless `COOKIE_SECURE=true`.
- **Errors carry their HTTP status** (`errors.AppError`). Handlers and services never pick status
  codes; `errors.Fail` is the only place that writes an error response.
- **5xx causes are logged, never serialized.** Clients get `"internal server error"`.
- **Health check pings the database** with a 2s timeout; the router depends on a small `Pinger`
  interface, so health tests need no database.
- **Gin 1.11 instead of 1.12**: 1.12 pulls the MongoDB BSON driver through `gin/binding`.
- **`gin.New()` + explicit middleware**, `SetTrustedProxies(nil)`, and server timeouts
  (`ReadHeaderTimeout` 10s, etc.) per the Gin skill.
- **Logs contain no bodies, cookies, or query strings** — method, path, status, duration, IP only.

### Tests

| Test | Proves |
|---|---|
| `TestLoadDefaults` | defaults are applied when env is empty |
| `TestLoadReportsAllMissingRequiredValues` | both missing required vars are reported together |
| `TestLoadRejectsShortSecret` | JWT secret shorter than 32 chars is rejected |
| `TestLoadRejectsInvalidOptionalValues` | bad TTL / bad bool are rejected |
| `TestProductionRequiresSecureCookie` | production requires `COOKIE_SECURE=true` |
| `TestHealthzOK` | healthy DB → 200 |
| `TestHealthzDatabaseDown` | DB failure → 503 + envelope |
| `TestUnknownRouteUsesErrorEnvelope` | 404 uses the envelope |
| `TestWrongMethodUsesErrorEnvelope` | 405 uses the envelope |
| `TestPanicIsRecoveredAsJSON` | panic → 500, internals not leaked |

### Verification

```
gofmt -l .        clean
go vet ./...      clean
go test ./...     all pass
GET /healthz      {"status":"ok"} 200          (real Postgres on 5432)
GET /nope         {"error":{"code":"not_found",...}} 404
POST /healthz     {"error":{"code":"method_not_allowed",...}} 405
SIGTERM           "shutdown signal received" → "server stopped", exit 0
missing secret    fatal: invalid configuration: - JWT_SECRET is required... exit 1
unreachable DB    fatal: ping database: ... connection refused, exit 1
```

---

## Task 3 — Auth module and session middleware ✅

### What exists

| File | Purpose |
|---|---|
| `internal/model/user.go` | `User` domain model (no JSON tags) |
| `internal/repository/user_repository.go` | `UserRepository` interface + pgx implementation; `23505` → `ErrEmailTaken` |
| `internal/util/password.go` | bcrypt cost 12, 72-byte cap, fixed dummy hash for timing |
| `internal/util/token.go` | HS256 issue/parse, `iss`/`sub`/`iat`/`exp`, algorithm pinned |
| `internal/util/context.go` | typed user-ID context helpers |
| `internal/service/auth_service.go` | `Register`, `Login`, `UserByID`; normalization, validation, timing equalisation |
| `internal/handler/auth_handler.go` | hardened cookie, 4 auth handlers |
| `internal/dto/auth.go` | request/response payloads |
| `internal/errors/http.go` | validator errors → field-level `400` |
| `internal/middleware/auth.go` | `RequireSession`: cookie → verified user ID |

### Endpoints

| Method | Path | Success | Failures |
|---|---|---|---|
| POST | `/api/auth/register` | `201 {user}` + `Set-Cookie` | `400`, `409 email_taken` |
| POST | `/api/auth/login` | `200 {user}` + `Set-Cookie` | `400`, `401` |
| POST | `/api/auth/logout` | `204` + cleared cookie | — |
| GET | `/api/auth/me` | `200 {user}` | `401` |

Cookie: `auth_token=<jwt>; Path=/; Max-Age=1800; HttpOnly; SameSite=Lax` (+ `Secure` when
configured). The token is never returned in a JSON body.

### Decisions

- **30-minute access token** (`.env`: `JWT_TTL=30m`), no refresh token.
- **JWT in the service, cookie in the handler.** The service returns `(user, token)`; only the
  transport layer sets cookies.
- **No duplicate-email pre-check.** `INSERT` relies on the `LOWER(email)` unique index; `23505`
  maps to `409`, so concurrent registrations cannot both succeed.
- **Timing-equalised login.** An unknown email still runs bcrypt against a fixed cost-12 hash.
- **Two validation layers.** Binding catches shape (required/min/max); the service trims and
  lowercases the email, validates it with `net/mail`, and checks the password in **bytes** (8–72)
  because bcrypt truncates past 72. Passwords are never trimmed.
- **`/me` re-reads the user from Postgres**, so a deleted account's token stops working immediately.
- **Explicit dependency injection in `cmd/server/main.go`** — the only place that knows every layer:
  `repository → service → handler → router`. `handler.NewRouter` receives already-built handlers and
  only mounts routes, so there is no DI container and no service locator.

### Tests

| Area | Coverage |
|---|---|
| password | hash ≠ plaintext, correct/wrong verify, 72 bytes accepted, 73 rejected |
| token | round-trip, expired, wrong secret, `alg=none`, tampered signature |
| service | email normalization, duplicate (case-insensitive), 4 validation cases, login success/failure/unknown, deleted user |
| handler | 201 + cookie flags, duplicate 409, validation 400, login 200/401, `/me` 200/401, forged cookie, logout clears cookie, malformed JSON |

### Verification

```
go vet ./...   clean
go test ./...  auth 4.4s, config and server packages pass
POST /api/auth/register  201, Set-Cookie: Path=/; Max-Age=1800; HttpOnly; SameSite=Lax
POST /api/auth/register (ALICE@example.com)  409 email_taken
POST /api/auth/login (wrong password)        401 invalid email or password
GET  /api/auth/me (cookie)                   200 {"user":{"id":1,"email":"alice@example.com"}}
POST /api/auth/register (short password)     400 fields.password
POST /api/auth/logout                        204, Set-Cookie: Max-Age=0
GET  /api/auth/me (after logout)             401 authentication required
DB check: email normalized, password_hash prefix $2a$12$, length 60
```

Note: the `go run ./cmd/server` process on :8080 must be restarted after the restructure.

---

## Task 4 — Hardening ✅

### What exists

| File | Purpose |
|---|---|
| `internal/middleware/security.go` | security headers, 64 KiB body cap, JSON-only bodies |
| `internal/errors/errors.go` | `413 payload_too_large`, `415 unsupported_media_type` |
| `internal/errors/http.go` | `*http.MaxBytesError` → `413` instead of a misleading `400` |
| `internal/handler/router.go` | middleware applied to every route |
| `internal/handler/security_test.go` | 8 tests (10 with subtests) |

### Behavior

| Case | Result |
|---|---|
| Any response | `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`, `Cache-Control: no-store` |
| `COOKIE_SECURE=true` | adds `Strict-Transport-Security: max-age=31536000; includeSubDomains` |
| Expired / tampered / `alg=none` cookie on `/me` | `401 unauthorized` |
| Body > 64 KiB | `413 payload_too_large` |
| `Content-Type` not `application/json` on POST | `415 unsupported_media_type` |
| `application/json; charset=utf-8` | accepted |

### Decisions

- **64 KiB cap** — the largest legitimate payload is a 2000-rune description (~8 KiB).
- **Headers on every response, HSTS only when `COOKIE_SECURE=true`** — local HTTP dev must not pin a browser to HTTPS.
- **`Cache-Control: no-store`** — `/api/auth/me` and task lists must not be cached by proxies.
- **JSON-only bodies** — the design already claimed this as a CSRF mitigation; `RequireJSON` makes it true. `SameSite=Lax` remains the primary defence.
- **Content-Length 0 skips the check**, so `POST /api/auth/logout` keeps working without a body.

### Verification

```
go vet ./...    clean
go test ./...   44 tests pass
GET  /healthz                             200 + 5 headers (no HSTS in dev)
GET  /api/auth/me expired|tampered|none   401 unauthorized
POST /api/auth/register 70 KiB            413 payload_too_large
POST /api/auth/register text/plain        415 unsupported_media_type
POST /api/auth/login                      200 + cookie (no regression)
GET  /api/auth/me cookie                  200
POST /api/auth/logout                     204
```

---

## Task 5 — Task module ✅

### What exists

| File | Purpose |
|---|---|
| `internal/model/task.go` | `Task` domain struct |
| `internal/dto/task.go` | create/update requests, `{task}` and `{tasks}` envelopes |
| `internal/repository/task_repository.go` | `TaskRepository` + pgx implementation; `ErrTaskNotFound` |
| `internal/service/task_service.go` | validation, status parsing, 404 translation |
| `internal/handler/task_handler.go` | 4 handlers + `parseTaskID` |
| `internal/handler/router.go` | `Deps` struct; `/api/tasks` behind `RequireSession` |
| `internal/service/task_service_test.go` | 11 tests |
| `internal/handler/task_handler_test.go` | 10 tests |

### Endpoints

| Method | Path | Success | Failures |
|---|---|---|---|
| GET | `/api/tasks?status=all\|active\|completed` | `200 {"tasks":[...]}` newest first | `400` bad status, `401` |
| POST | `/api/tasks` | `201 {"task":{...}}` | `400` validation, `401` |
| PATCH | `/api/tasks/:id` | `200 {"task":{...}}` | `400` empty patch/invalid id, `401`, `404` |
| DELETE | `/api/tasks/:id` | `204` | `400` invalid id, `401`, `404` |

### Decisions

- **`handler.Deps` struct** replaces positional router arguments now that two modules exist; two
  same-typed `*Handler` parameters would be silently swappable.
- **Ownership is a SQL predicate**, not a lookup: every query carries `WHERE user_id = $1`, and
  `pgx.ErrNoRows` / `RowsAffected() == 0` → `404`. A foreign task and a missing task are identical.
- **PATCH uses pointers** (`*string`, `*bool`) so `{"completed":false}` differs from "absent".
  `UPDATE … SET title = COALESCE($3, title)` leaves untouched columns alone; `description:""` clears.
- **Empty patch → `400 validation_failed`** with `fields.body`; the DB is never touched.
- **Rune counts** (`utf8.RuneCountInString`) match Postgres `char_length` and the `CHECK` constraints.
- **`updated_at = now()` in SQL**, one clock, so `createdAt`/`updatedAt` cannot drift.
- **Empty list serialises as `[]`**, never `null`, so the frontend can map without a guard.
- **Status filter** parsed in the service: `""`/`all` → no filter, `active`/`completed` → `*bool`,
  anything else → `400 fields.status`.

### Verification

```
go vet ./...    clean
go test ./...   65 tests pass

alice: POST /api/tasks               201, title trimmed, completed=false
alice: GET  /api/tasks               200, newest first
alice: PATCH completed=true          200, updatedAt advanced
alice: status=active|completed       1 task each; status=archived → 400
alice: POST blank title              400 fields.title
alice: PATCH {}                      400 fields.body
alice: DELETE /api/tasks/abc         400 invalid_id
bob:   GET   /api/tasks              200 {"tasks":[]}
bob:   PATCH/DELETE alice's task     404 (task untouched)
none:  GET   /api/tasks              401
alice: DELETE own task               204, then 404
psql:  tasks scoped by user_id, only the expected row remains
```

---

## Task 6 — Backend integration suite ✅

### What exists

| File | Purpose |
|---|---|
| `backend/internal/integration/api_test.go` | `TestMain` (test-DB guard + schema reset), `TestFullFlowTwoUsers`, `TestDuplicateEmailHitsUniqueIndex` |
| `Makefile` | `TEST_DATABASE_URL` var, `test-integration` target (creates `todo_test` on first use) |
| `README.md` | Tests section: `make test` vs `make test-integration` |
| `docs/design.md` | task-6 row points at `internal/integration` |

### Decisions

- **The suite lives in `internal/integration`, not `internal/handler`**, because it wires the real
  repository/service/handler stack; the handler tests keep using the in-memory fake.
- **Dedicated `todo_test` database**, never the dev data. `TestMain` refuses any URL whose database
  name does not end in `_test` before it drops anything.
- **Schema reset is done by the test binary**, not golang-migrate: `DROP TABLE IF EXISTS tasks, users
  CASCADE`, then every `migrations/*.up.sql` in sorted order. Reruns are deterministic.
  `shortcut:` if migrations ever need down/ordering semantics, switch to golang-migrate here.
- **No `TEST_DATABASE_URL` → skip with exit 0**, so `go test ./...` and `make test` need no Docker.
  A URL that is set but unreachable fails loudly instead of silently skipping.
- **`httptest.NewServer` + stdlib `cookiejar`** gives real cookie round-trips (set/expire/scope)
  rather than hand-copied `Cookie` headers.
- **The test router is wired like `cmd/server/main.go`** — real repositories, services, token
  manager and 30-minute TTL — so the suite covers the composition root too.
- **The DB is asserted directly** at the end of the flow (`SELECT count(*) … WHERE user_id = $1`),
  so isolation is proven in storage, not only in responses.

### Verification

```
DOCKER_CONFIG=… make test-integration   # first run creates todo_test
--- PASS: TestFullFlowTwoUsers (0.44s)
--- PASS: TestDuplicateEmailHitsUniqueIndex (0.42s)
ok   …/internal/integration  0.884s

make test-integration (second run)      # same result: reset is idempotent
go test ./...                           # integration package prints "skipped", 65 unit tests pass
go vet ./...                            clean
TEST_DATABASE_URL=<...>/todo go test ./internal/integration/...   # refuses: "todo" does not end in _test
```

Flow covered: register two users → `/me` (and `401` anonymous) → create/list with newest-first
order → `PATCH completed` with `updatedAt` advanced → `?status=active|completed` → blank title,
empty patch and bad status all `400` → cross-user `PATCH`/`DELETE` both `404` with the owner's row
unchanged → delete `204` then `404` → logout `204` then `/me` `401` → direct SQL row counts.

---

## Task 7 — Frontend foundation ✅

### What exists

| File | Purpose |
|---|---|
| `frontend/package.json` | React 19.3, Vite 7.3, TypeScript 5.9 (strict), Geist via `@fontsource-variable/geist` |
| `frontend/tsconfig.json` | `strict`, `noUncheckedIndexedAccess`, `verbatimModuleSyntax`, `noEmit` |
| `frontend/vite.config.ts` | `/api` → `http://localhost:8080` proxy, in `dev` and `preview` |
| `frontend/index.html` | `lang`, `color-scheme`, `theme-color` matching the ground |
| `frontend/src/main.tsx` | `createRoot` + `AuthProvider` |
| `frontend/src/App.tsx` | shell that renders the real session state (screens replace it in Tasks 8–9) |
| `frontend/src/types.ts` | `User`, `Task`, `TaskFilter`, request inputs, error envelope |
| `frontend/src/api/client.ts` | `request<T>()` + `ApiError` (`status`, `code`, `fields`) |
| `frontend/src/api/auth.ts` | `me`, `login`, `register`, `logout` |
| `frontend/src/auth/AuthContext.tsx` | `loading \| authenticated \| anonymous`, actions, `clearSession` |
| `frontend/src/styles.css` | dealt tokens, reset, shell, one motion moment |
| `PRODUCT.md` | product truth (users, purpose, constraints, principles) |

### Design direction

Dealt with `design-taste-frontend` (`draw.mjs --app --kind workspace --screens home,list`), chosen from
the two options: **achromatic-ledger**, light, sparse.

- **Move:** no colour in the chrome. Primary action near-black; every figure in ink; one hue
  (`#1d5fbf`) only as a small state signal.
- **Face:** the stone ground (`#f4f3f1`) from the product's world — a shop's shelf, one step off white —
  with white panels and the product's own object (a ticked box) as the mark.
- **Type:** Geist, self-hosted through Fontsource (no CDN, so the bundle stays self-contained). Sizes
  12/14/16/24 px, one weight step (400/500).
- **Radii:** 8 px cards, 6 px controls, per the sheet.
- **Rejected:** `editorial-greeting` — a greeting-as-h1 home is the studied template, and a to-do list
  is worked through rather than greeted.

### Decisions

- **No router yet.** `react-router-dom` arrives in Task 8 with the pages that use it; installing it now
  would ship an unused dependency.
- **One `request<T>()`, no client library and no react-query.** Three screens and a cookie session do
  not justify either; `ApiError` carries the backend's `fields` so forms can render them verbatim.
- **`401` is not handled globally.** `AuthContext` treats it as the normal signed-out case during
  bootstrap; Task 9 will call `clearSession()` when a task call 401s.
- **`credentials: 'same-origin'`** is explicit, and the proxy keeps the cookie first-party in dev and
  preview — the same arrangement a reverse proxy provides when deployed.
- **Fonts are bundled, not fetched.** A CDN link would break offline use and add a render-blocking third
  party.
- **`preview` gets the same proxy as `dev`** so the production-build design gate runs against real
  session state instead of a network error.

### Verification

```
npm run typecheck   clean
npm run build       built in 960ms; 225 kB JS (70 kB gzip), 4.4 kB CSS, Geist latin 29 kB

curl localhost:5173/api/auth/me → 401 {"error":{"code":"unauthorized",…}}   (proxy → Go API)
login through :5173 → 200, cookie stored; GET /api/auth/me → 200 alice@example.com
logout → 204; GET /api/auth/me → 401                                        (cookie round-trip)

lint.mjs --app frontend/src           0 FAIL, 0 WARN
inspect.mjs --app (production build)  0 FAIL, 2 WARN
impeccable detect (1280 and 390)      0 findings
contrast                              muted/ground 4.81, muted/panel 5.33, text 17.85, signal 6.10
```

The two inspector warnings are the placeholder, not the layout: `hero-void` and `unused-canvas` measure
the short centred column at 1440px. The centred 640 px column is the sheet's "personal home" canvas;
the task list fills it in Task 9, which is when the gate must be re-run.

---

## Task 8 — Auth UI ✅

### What exists

| File | Purpose |
|---|---|
| `frontend/src/App.tsx` | route table: `/login`, `/register`, protected `/tasks`, `*` → `/tasks` |
| `frontend/src/main.tsx` | `BrowserRouter` around `AuthProvider` |
| `frontend/src/components/ProtectedRoute.tsx` | `loading` → gate, `anonymous` → `/login`, else `<Outlet/>` |
| `frontend/src/components/SessionGate.tsx` | “Checking your session…” while `/api/auth/me` is in flight |
| `frontend/src/components/AuthForm.tsx` | shared form: fields, request, server field errors, focus, pending label |
| `frontend/src/components/AppHeader.tsx` | wordmark + signed-in email + Sign out; reused by Task 9 |
| `frontend/src/pages/Login.tsx`, `Register.tsx` | card, heading, link to the other screen; signed-in users bounce to `/tasks` |
| `frontend/src/pages/TasksPage.tsx` | placeholder list screen (Task 9 replaces the body) |
| `frontend/src/styles.css` | header, auth card, fields, errors, button, gate; `--border-strong`, `--negative`, `--negative-bg`, `--accent-hover`, `--measure-form` |
| `frontend/package.json` | `react-router-dom` 7.18 — the only new dependency |

### Decisions

- **One protected route, so no `state.from` dance.** Anonymous users always land on `/login`; both auth
  pages redirect an authenticated user to `/tasks`. A “return to where you were” round trip buys nothing
  with a single protected page.
- **No flash of the wrong screen.** `loading` renders the gate, not a redirect, so a valid cookie never
  shows the login form for a frame.
- **Native validation stays on, the server stays authoritative.** `required`, `type="email"` and
  `minLength={8}` on register give instant feedback; the backend's `fields` still render inline, because
  the service normalises and validates more than the browser does.
- **Server copy is shown verbatim.** The backend already writes user-facing messages, so the UI never
  invents a second vocabulary. A `409 email_taken` carries no `fields`; the form attaches its message to
  the email input, which is the only thing the user can change.
- **Focus moves to the first invalid field**, and `aria-invalid` / `aria-describedby` tie the message to
  it. Form-level failures (wrong password, network) use `role="alert"`.
- **Sign out is local-first.** `logout()` clears context state even if the request fails, so the guard
  redirects to `/login` without a second navigation call.
- **No confirm-password field, no reveal toggle, no “remember me”.** None are in the brief; the backend
  has no such concepts, and the session cookie is already 30 minutes.

### Design

Same dealt sheet (`achromatic-ledger`), no redraw: stone ground, one white 400 px card, ink labels,
near-black primary button, one hue reserved for focus and the session signal.

One token added for a measured reason: `--border-strong` (`#8a8a8a`, **3.45:1** on white) for form
boundaries, because the sheet's decorative `--border` is 1.26:1 and fails the 3:1 non-text contrast that
an input outline needs. Errors use `--negative` (`#b42318`, 6.57:1 on the card, 6.08:1 on its own tint).

### Verification

Real browser, driven over CDP (throwaway script in `/tmp`, no new dependency) against the dev server:

```
20/20 checks passed
anonymous /tasks            → /login
wrong password              → stays on /login, "invalid email or password" at form level
register (fresh email)      → /tasks, email in the header
reload                      → still /tasks (HttpOnly cookie survives)
signed-in /login            → /tasks
Sign out                    → /login, and /tasks is protected again
duplicate email             → inline on #email, aria-invalid, focus on #email
short password              → inline on #password, hint + error in aria-describedby
correct login               → /tasks
no page exceptions

npm run typecheck           clean
npm run build               268 kB JS (85 kB gzip), 6.9 kB CSS
lint.mjs --app frontend/src 0 FAIL, 0 WARN
inspect.mjs --app (build)   /login and /register: 0 FAIL, 1 WARN (hero-void)
impeccable detect           /login and /register, 1280 and 390: 0 findings
contrast                    border 3.45 · error 6.57/6.08 · muted 5.33/4.81 · text 19.80 · focus 6.10
```

The remaining warning is `hero-void`: at 1440 px a centred 400 px card leaves equal empty margins. That
is the deliberate composition for a single-action screen — the sheet's own arrangement — not an unfilled
layout. `/tasks` is inspected in Task 9, when it carries a real screen; today it is a placeholder and
needs a session to be seen at all.

---

## Task 9 — Tasks UI ✅

### What exists

| File | Purpose |
|---|---|
| `frontend/src/api/tasks.ts` | typed `list` / `create` / `update` / `remove` wrappers over the shared client |
| `frontend/src/tasks/useTasks.ts` | list state, loading/error state, mutations, 401 → `clearSession()` |
| `frontend/src/components/TaskForm.tsx` | one form for create and edit: field errors, Escape to cancel, focus handling |
| `frontend/src/components/TaskItem.tsx` | checkbox, title button (opens edit), optional note, delete button |
| `frontend/src/components/FilterTabs.tsx` | `All \| Active \| Completed`, `aria-pressed`, labelled group |
| `frontend/src/pages/TasksPage.tsx` | page head + counts, composer, filters, list, loading/empty/error states |
| `frontend/src/styles.css` | task screen, rows, filters, states, skeleton, 44 px coarse-pointer targets |

### Behavior

- **Create** — title required (255 max), note optional (2000 max). New tasks appear first, the form
  clears, and focus stays in the title field for the next entry.
- **Toggle** — checkbox PATCHes `{completed}`; the row is struck through and muted, counts update.
- **Edit** — clicking the title swaps the row for the same form, prefilled. Save sends only the keys that
  changed; Escape or Cancel restores the row without a request; focus returns to the title button.
- **Delete** — `window.confirm` first, then `DELETE`; focus returns to the add field. A `404` means the row
  is already gone, so it is removed without an error.
- **Filters** — instant, client-side, with honest counts from the full list. Each filter has its own empty
  copy: first run, no active tasks, nothing completed.
- **Failure** — a failed load shows an in-place message with **Try again**; a failed mutation shows a
  dismissible notice above the list and leaves the row unchanged. A `401` clears the session and the
  guard sends the user to `/login`.

### Decisions

- **One `GET /api/tasks`, filtered in the browser.** The list is small and unpaginated, so tabs switch
  without a round trip and the counts can never disagree with the visible rows. The server-side
  `?status=` filter stays implemented and covered by the integration suite; the UI does not need it.
- **Server-first mutations, no optimistic writes.** Every mutation waits for the API response and replaces
  the row with what the server returns. A rejected write never leaves a row that looks saved.
- **Partial PATCH.** The backend's pointer-based update treats an absent key as “unchanged”, so the UI
  sends only edited fields; a note-only edit cannot accidentally rewrite the title.
- **No new dependency.** State is local React; confirmation is the platform's `window.confirm`; nothing
  was added for forms, focus, or data fetching.
- **Description is a single-line note.** It is a short annotation under the title, not a second document;
  the API still accepts multi-line text, but the composer is one input.
- **Counts appear once**, under the heading (`3 left · 1 done`); the filter tabs are labels, not counters,
  so the same number is not repeated three times.

### Design

Same dealt sheet (`achromatic-ledger`): stone ground, one white 640 px card, ink text, one hue for state
and focus. The page head sits outside the card so the screen has a title before the tool; the composer,
filters and rows share the card, separated by hairlines rather than nested boxes. Completed rows recede
with a strike and muted colour; the only accent is the checkbox and focus ring.

No new tokens. Control targets grow to 44 px on coarse pointers; the delete control is a 32 px hit area on
fine pointers and 44 px on touch. Focus is visible on every control, and the title button's ring hugs the
text instead of spanning the row, so a focused row does not read as an input.

### Verification

Real browser over CDP (throwaway script in `/tmp`) against the dev server, with a fresh account:

```
44/44 checks passed
register                    → /tasks, empty state, 0 left · 0 done
create title only           → row listed, counts move, form clears, focus stays
create with note            → newest first, note rendered
toggle                      → strike-through, completed flag, counts move
filters                     → All / Active / Completed show the right rows, aria-pressed
edit                        → prefilled, title/note saved, counts unchanged
partial PATCH               → {"title":…,"description":…}, then {"description":…} only
Escape                      → closes the form, sends no request, title unchanged
delete cancelled            → row stays; confirmed → row gone, focus to add field
filtered empty              → "No active tasks. Everything is done."
load failure                → in-place error + Try again; retry loads the list
expired session             → 401 sends the user to /login
no page exceptions

npm run typecheck           clean
npm run build               274.92 kB JS (87.01 kB gzip), 9.90 kB CSS
lint.mjs --app frontend/src 0 FAIL, 0 WARN
inspect.mjs --app (build)   1440 / 768 / 390: 0 FAIL, 0 WARN
impeccable detect           1280 and 390, signed in: 0 findings
contrast                    muted/ground 4.81 · muted/panel 5.33 · text 17.85–19.80 · signal 6.10
```

The earlier `unused-canvas` warning at 1440 px is gone: the task card now fills the centred column with
real content. The `hero-void` warning belonged to the auth screens and does not apply here.

---

## Task 10 — Docs and final verification ✅

### What exists

- **README** — final quick start (PostgreSQL via Compose, API and Vite on the host), test targets, routes,
  layout, and the delivery scope.
- **`docs/design.md`** — status set to implemented; the project tree and delivery table match the files
  that actually exist; dev architecture shown as Vite proxying `/api`.
- **`docs/implementation-report.md`** — every task closed, verification evidence, limitations, and this
  close-out.
- **Backend suite** — `go test ./...` passes; the integration package skips cleanly when
  `TEST_DATABASE_URL` is unset.

### Scope

The required assignment features are delivered: registration, login/logout, protected task CRUD,
complete/active toggle, All/Active/Completed filters, PostgreSQL persistence, and per-user ownership.
Local development uses Compose for PostgreSQL and migrations, then `make api` and `make web` for the app.

Container images for the API and frontend, nginx, and frontend unit tests were outside the submitted
scope. They are listed here rather than silently claimed, so the repository matches its docs.

### Verification

```
go test ./...               all packages pass (integration skips without TEST_DATABASE_URL)
npm run typecheck           clean
npm run build               274.92 kB JS (87.01 kB gzip), 9.90 kB CSS
browser checks              auth 20/20, tasks 44/44
design gates                lint 0 FAIL/0 WARN · inspect 0 FAIL/0 WARN · impeccable 0 findings
```

---

## Review notes and deviations

- **Rejected AI suggestion:** the `golang-gin-api` skill recommends `html.EscapeString` on inbound
  strings. Not applied — escaping on write corrupts data and double-encodes. XSS is handled at
  render time by React; the API stores raw text and JSON-encodes it.
- **Module path** is `github.com/tinkhmt/shopback-todo/backend` (assumed GitHub handle). Rename with
  `go mod edit -module <path>` if the repository lives elsewhere.
- **Docker credentials on this machine:** `~/.docker/config.json` references a missing
  `docker-credential-desktop`. Verification used a throwaway `DOCKER_CONFIG`. No global config was
  modified.

## Known limitations (deliberate, see design.md)

No rate limiting, refresh tokens, email verification, password reset, pagination, task
ordering/drag-drop, multi-device logout, HTTPS termination, or server-side token revocation. Logout
clears the cookie; the JWT remains valid until `exp` (30 min). There are no frontend unit tests and no
container images for the API or frontend; run the app from source with the Makefile targets.

## How to run

```bash
cp .env.example .env
make db-up && make migrate-up
make api            # API on :8080
make test           # backend tests
make test-integration  # full API flow against a real todo_test database
make web            # frontend dev server on :5173, proxies /api to :8080
```

## Change log

| Date | Change |
|---|---|
| 2026-11-01 | Task 1: Compose, migrations, schema verified |
| 2026-11-01 | Task 2: backend foundation, error envelope, health check, tests |
| 2026-11-01 | Task 3: auth module, 30-minute JWT cookie, session middleware, tests |
| 2026-11-01 | Refactor: layer-first layout (`dto`, `errors`, `handler`, `logger`, `model`, `repository`, `service`, `util`), `cmd/server`, composition root |
| 2026-11-01 | Task 4: security headers, 64 KiB body cap, JSON-only bodies, token edge-case tests |
| 2026-11-01 | Task 5: task CRUD, status filter, SQL-enforced ownership, 21 tests |
| 2026-11-01 | Task 6: Postgres integration suite (two users, full flow, DB-level isolation), `make test-integration` |
| 2026-11-01 | Task 7: frontend foundation — Vite/TS, typed API client, `AuthContext`, dealt design tokens (`achromatic-ledger`) |
| 2026-11-01 | Task 8: auth UI — routes, `ProtectedRoute`, login/register forms with server field errors, sign-out header |
| 2026-11-01 | Task 8 follow-up: product wordmark (mark + name) on the login and register screens, matching the tasks header |
| 2026-11-01 | Task 9: tasks UI — list, create, inline edit, complete, delete, filters, loading/empty/error states |
| 2026-11-01 | Task 10: docs and final verification — statuses closed, scope notes, README final |
