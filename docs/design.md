# ShopBack To-Do — Design & Delivery Plan

Status: **proposed — awaiting review**
Scope: small take-home. Optimised for clarity and reviewability, not production scale.

## 1. Requirement analysis

### Functional
| # | Requirement | Acceptance |
|---|---|---|
| 1 | Register | Valid email + password creates a user; duplicate email → `409` |
| 2 | Login / logout | Correct credentials set an HttpOnly cookie; logout clears it |
| 3 | Protected task management | Every `/api/tasks*` route requires a valid cookie |
| 4 | Task CRUD | Create, list, edit, delete; edits are partial (`PATCH`) |
| 5 | Complete / active | Toggle `completed` via `PATCH` |
| 6 | Filter | `GET /api/tasks?status=all\|active\|completed` |
| 7 | Persistence | PostgreSQL, applied by golang-migrate migrations |
| 8 | Ownership | All task queries are scoped by `user_id`; foreign task → `404` |

### Non-functional
- Backend: modular monolith, layer-first packages (`handler → service → repository`) under `internal/`.
- Consistent JSON error envelope, backend-only validation of every input.
- Config exclusively from environment variables; app fails fast on missing/weak secrets.
- Frontend: strict TypeScript, no `any` in application code, typed API boundary.
- One-command local stack: `docker compose up --build`.

### Explicitly out of scope (named so they are not accidental omissions)
Email verification, password reset, refresh tokens, rate limiting, pagination, task ordering/drag-drop, multi-device logout, HTTPS termination. Each is listed in the README as a production follow-up.

## 2. Architecture

```
Browser ──HTTP──> nginx (SPA + /api proxy) ──> Gin router
                                                   │
                                    middleware: recover, request log, auth
                                                   │
                                     handler → service → repository
                                                   │
                                               pgxpool
                                                   │
                                              PostgreSQL
```

Auth and tasks share the same layer packages; each handler/service/repository pair is a separate
type, so features stay isolated without per-feature directories.

**Same-origin by design.** In dev, Vite proxies `/api` → `:8080`; in the container, nginx does the same. The browser therefore never makes a cross-origin request: no CORS middleware, no wildcard, no preflight surface. Cookies are `SameSite=Lax`, `HttpOnly`, `Secure` when `COOKIE_SECURE=true`.

### Project structure

```
.
├── docker-compose.yml          # db, migrate, backend, frontend
├── Makefile                    # dev shortcuts
├── .env.example
├── docs/
│   ├── design.md               # this file
│   └── AI_USAGE.md             # what AI wrote, what I verified (Task 10)
├── backend/
│   ├── cmd/server/main.go      # composition root + graceful shutdown
│   ├── internal/
│   │   ├── config/             # env parsing, fail-fast
│   │   ├── database/           # pgxpool
│   │   ├── dto/                # request/response payloads
│   │   ├── errors/             # AppError + single JSON writer
│   │   ├── handler/            # HTTP handlers + router assembly
│   │   ├── logger/             # slog setup
│   │   ├── middleware/         # session auth, request logging, panic recovery
│   │   ├── model/              # domain structs
│   │   ├── repository/         # pgx implementations
│   │   ├── service/            # business rules
│   │   └── util/               # bcrypt, JWT, context helpers
│   ├── migrations/             # 000001_init.up.sql / .down.sql
│   └── Dockerfile
└── frontend/
    ├── src/
    │   ├── api/                # client.ts, auth.ts, tasks.ts
    │   ├── types.ts            # User, Task, TaskFilter, ApiError
    │   ├── auth/AuthContext.tsx
    │   ├── pages/              # Login, Register, Tasks
    │   ├── components/         # TaskForm, TaskItem, FilterTabs, ProtectedRoute
    │   └── styles.css
    ├── vite.config.ts          # /api proxy
    ├── nginx.conf
    └── Dockerfile
```

### Data model

```sql
users(id BIGINT IDENTITY PK, email TEXT NOT NULL, password_hash TEXT NOT NULL,
      created_at TIMESTAMPTZ NOT NULL DEFAULT now())
UNIQUE INDEX users_email_lower_key ON users (LOWER(email));

tasks(id BIGINT IDENTITY PK, user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      title TEXT NOT NULL CHECK (char_length(title) BETWEEN 1 AND 255),
      description TEXT NOT NULL DEFAULT '' CHECK (char_length(description) <= 2000),
      completed BOOLEAN NOT NULL DEFAULT false,
      created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
      updated_at TIMESTAMPTZ NOT NULL DEFAULT now())
INDEX tasks_user_created_idx ON tasks (user_id, created_at DESC, id DESC);
```

`BIGINT IDENTITY` (not UUID): IDs are not secrets and ownership is enforced in SQL, so opacity buys nothing. One index covers the only list query (`WHERE user_id = ? ORDER BY created_at DESC`); the status filter is a cheap residual predicate at this scale.

### API contract

All bodies are JSON. Errors: `{"error":{"code","message","fields"?}}`.

| Method | Path | Body | Success |
|---|---|---|---|
| POST | `/api/auth/register` | `{email,password}` | `201 {user}` + cookie |
| POST | `/api/auth/login` | `{email,password}` | `200 {user}` + cookie |
| POST | `/api/auth/logout` | — | `204` + cleared cookie |
| GET | `/api/auth/me` | — | `200 {user}` |
| GET | `/api/tasks?status=` | — | `200 {tasks:[]}` |
| POST | `/api/tasks` | `{title,description?}` | `201 {task}` |
| PATCH | `/api/tasks/:id` | `{title?,description?,completed?}` | `200 {task}` |
| DELETE | `/api/tasks/:id` | — | `204` |
| GET | `/healthz` | — | `200` |

`PATCH` (not `PUT`) because the toggle sends only `{completed:true}`. An empty patch body is `400`.

## 3. Security concerns & edge cases

| Concern | Mitigation |
|---|---|
| Password storage | bcrypt cost 12; min 8 chars, max 72 bytes (bcrypt truncates silently beyond that) |
| User enumeration | Register duplicate → `409`; login always `invalid email or password`; compare against a dummy hash when the user is absent |
| JWT forgery | HS256 only, algorithm asserted on parse; `JWT_SECRET` required and ≥32 bytes; `exp` required |
| XSS token theft | Token never touches JS: `HttpOnly` cookie, no `localStorage` |
| CSRF | `SameSite=Lax` + mutations are non-GET + JSON-only bodies + same-origin proxy |
| IDOR | Every read/write has `user_id = $1`; foreign resource returns `404`, not `403` (no existence leak) |
| SQL injection | pgx parameterised queries only; no string-built SQL |
| Input abuse | Email format/length, title 1–255 runes after trim, description ≤2000 runes, `status` enum, request body size cap |
| Error leakage | Internal errors logged with `slog`, client gets a generic `500`; DB errors never serialised |
| Log hygiene | Passwords, hashes and tokens never logged |
| Race on register | DB unique index is the source of truth; map pgx `23505` → `409` |
| ID parsing | Non-numeric `:id` → `400`; valid-but-missing → `404` |
| Expired session mid-use | Frontend maps `401` → clear auth state → redirect to login |
| Logout semantics | Stateless JWT: logout clears the cookie, token stays valid until `exp` (30 min). Documented trade-off; `token_version` column is the upgrade path |

## 4. Delivery tasks (one at a time, each independently runnable)

| # | Task | Files | Verification |
|---|---|---|---|
| 1 | Skeleton + DB | `.gitignore`, `.env.example`, `docker-compose.yml` (db, migrate), `Makefile`, `backend/migrations/*`, README stub | `make db-up migrate-up`, inspect `\d users`, `\d tasks`; down/up round-trip |
| 2 | Backend foundation | `go.mod`, `cmd/server/main.go`, `internal/{config,database,errors,logger,handler,middleware}` | `/healthz` 200, unknown route → JSON 404, missing `JWT_SECRET` → fail fast |
| 3 | Auth module + session middleware | `internal/{model,repository,service,handler,dto,util,middleware}`, tests | register/login/me/logout via curl; `go test ./internal/service/... ./internal/handler/...` |
| 4 | Hardening + token edge cases | `internal/middleware/security.go`, body-size cap, handler tests | `alg=none`/tampered/expired → 401, security headers present |
| 5 | Task module | `internal/{model,repository,service,handler,dto}` task files + tests | CRUD, filter, cross-user `404`, validation `400` |
| 6 | Integration suite | `internal/handler/router_test.go`, `make test-integration` | two users, full flow, isolation proven against real Postgres |
| 7 | Frontend foundation | `package.json`, `tsconfig*`, `vite.config.ts`, `src/{main,App,types,styles}`, `src/api/*`, `src/auth/AuthContext.tsx` | `npm run dev`, typed client compiles under `tsc --noEmit` |
| 8 | Auth UI | `pages/{Login,Register}`, `components/ProtectedRoute.tsx` | register → tasks, refresh keeps session, logout → login, direct `/tasks` redirects |
| 9 | Tasks UI | `pages/TasksPage.tsx`, `components/{TaskForm,TaskItem,FilterTabs}`, `api/tasks.ts` | CRUD, toggle, filters, loading/empty/error states |
| 10 | Tests, images, docs | Vitest setup + frontend tests, both `Dockerfile`s, `nginx.conf`, full compose, `README.md`, `docs/AI_USAGE.md` | `docker compose up --build` fresh run; `make test` |

Skills in use: `golang-testing`, `golang-security`, `auth-implementation-patterns`, `postgresql-table-design`, `docker-compose-patterns`, `tdd`, plus `design-taste-frontend` / `web-design-guidelines` for the UI tasks.

## 5. Key decisions (trade-offs)

1. **No CORS** — same-origin proxy in both dev and prod removes a whole class of misconfiguration.
2. **No ORM** — hand-written SQL in repositories keeps pgx idiomatic and queries reviewable.
3. **No migration code in the binary** — golang-migrate runs as a compose service, so the API container has no schema privileges.
4. **No state library / react-query** — `AuthContext` + local state is enough for three screens.
5. **30-minute access token, no refresh token** — a stolen cookie is useless after 30 minutes and the
   implementation stays small. Users re-authenticate when it expires; refresh rotation is the upgrade
   path if the brief ever requires longer sessions.
5. **bcrypt over argon2** — stdlib-adjacent (`x/crypto`), no parameter tuning, expected by reviewers.
