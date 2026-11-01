# Implementation Report

Living document: updated after each completed task. Architecture and rationale live in
[design.md](design.md); this file records **what exists, what was verified, and what changed**.

Last updated: 2026-11-01 · Status: **Task 3 of 10 complete**

## Progress

| # | Task | Status |
|---|------|--------|
| 1 | Skeleton, Compose, database migrations | ✅ Done |
| 2 | Backend foundation (config, DB, router, errors) | ✅ Done |
| 3 | Auth module (register / login / logout / me) | ✅ Done |
| 4 | Hardening + token edge cases | ⬜ Next |
| 5 | Task module (CRUD, filter, ownership) | ⬜ Pending |
| 6 | Backend integration test suite | ⬜ Pending |
| 7 | Frontend foundation (Vite, types, API client) | ⬜ Pending |
| 8 | Auth UI | ⬜ Pending |
| 9 | Tasks UI | ⬜ Pending |
| 10 | Frontend tests, Docker images, docs | ⬜ Pending |

## Environment

| Item | Value |
|---|---|
| Go | 1.25+ required (pgx v5.12 floor); verified with 1.26.5 |
| Node | 22 (frontend, later tasks) |
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

No rate limiting, refresh tokens, email verification, password reset, pagination, or server-side
token revocation. Logout clears the cookie; the JWT remains valid until `exp` (30 min).

## How to run

```bash
cp .env.example .env
make db-up && make migrate-up
make api            # API on :8080
make test           # backend tests
```

## Change log

| Date | Change |
|---|---|
| 2026-11-01 | Task 1: Compose, migrations, schema verified |
| 2026-11-01 | Task 2: backend foundation, error envelope, health check, tests |
| 2026-11-01 | Task 3: auth module, 30-minute JWT cookie, session middleware, tests |
| 2026-11-01 | Refactor: layer-first layout (`dto`, `errors`, `handler`, `logger`, `model`, `repository`, `service`, `util`), `cmd/server`, composition root |
