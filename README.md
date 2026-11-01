# ShopBack To-Do

A small full-stack To-Do application: React + TypeScript (Vite) on the frontend, Go + Gin on the
backend, PostgreSQL for persistence, JWT in an HttpOnly cookie for authentication.

> Status: complete — all required features are implemented and verified. The plan and architecture live in
> [docs/design.md](docs/design.md); what was built and verified lives in
> [docs/implementation-report.md](docs/implementation-report.md).

## Stack

| Layer      | Choice                                          |
| ---------- | ----------------------------------------------- |
| Frontend   | React, TypeScript, Vite                         |
| Backend    | Go, Gin, pgx                                    |
| Database   | PostgreSQL 17                                   |
| Auth       | JWT (HS256) in an HttpOnly, SameSite=Lax cookie |
| Migrations | golang-migrate                                  |
| Local dev  | Docker Compose                                  |

## Prerequisites

- Docker with Compose v2 (`docker compose version`)
- Go 1.25+ and Node 20+ for running the services outside containers

## Quick start

```bash
cp .env.example .env
make db-up          # start PostgreSQL and wait for the healthcheck
make migrate-up     # apply migrations with golang-migrate
make api            # run the API on :8080 against the local database
```

Check it is alive:

```bash
curl localhost:8080/healthz     # {"status":"ok"}
curl localhost:8080/nope        # 404 in the shared error envelope
```

PostgreSQL is exposed on `${POSTGRES_PORT}` (default `5432`). Inspect the schema:

```bash
docker compose exec db psql -U todo -d todo -c '\d users' -c '\d tasks'
```

## Tests

```bash
make test              # unit + handler tests, no database needed
make test-integration  # full API flow against a dedicated todo_test database
```

`make test-integration` creates `todo_test` on first use and resets its schema on every run, so it
never touches development data.

## Frontend

```bash
make api            # terminal 1: the API on :8080
make web            # terminal 2: installs on first run, serves :5173
```

Vite proxies `/api` to the API, so the browser only ever talks to one origin and the session cookie
stays first-party — the same arrangement a reverse proxy provides when deployed.

```bash
cd frontend
npm run typecheck   # strict TypeScript, no emit
npm run build       # typecheck + production bundle
```

Routes: `/login`, `/register`, and `/tasks` (protected — an anonymous visitor is sent to `/login`). The
tasks screen adds, edits inline, completes, deletes and filters tasks; counts and empty/error states are
part of the same screen.

Other targets: `make help`.

## Layout

```
backend/
  cmd/server/     entry point (composition root), graceful shutdown
  internal/
    config/       environment parsing and validation
    database/     pgx connection pool
    dto/          request/response payloads
    errors/       typed application errors + JSON envelope
    handler/      HTTP handlers and router assembly
    logger/       slog setup
    middleware/   session auth, request logging, panic recovery
    model/        domain structs
    repository/   pgx implementations
    service/      business rules
    util/         bcrypt, JWT, context helpers
  migrations/     golang-migrate SQL migrations
frontend/
  src/api/        typed client, auth and task calls
  src/auth/       AuthContext (session bootstrap, login, logout)
  src/tasks/      useTasks (list state, mutations, load/action errors)
  src/components/ AuthForm, AppHeader, ProtectedRoute, SessionGate, TaskForm, TaskItem, FilterTabs
  src/pages/      Login, Register, TasksPage
  src/types.ts    wire types mirroring the Go DTOs
  src/styles.css  design tokens, reset, screens
docs/             design notes and implementation report
```

`docker-compose.yml` defines PostgreSQL and a one-shot `migrate` service. The API and frontend run on the
host with `make api` and `make web`; container images for the app services are not part of this delivery.
