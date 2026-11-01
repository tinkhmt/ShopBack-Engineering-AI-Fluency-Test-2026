# ShopBack To-Do

A small full-stack To-Do application: React + TypeScript (Vite) on the frontend, Go + Gin on the
backend, PostgreSQL for persistence, JWT in an HttpOnly cookie for authentication.

> Work in progress. The full plan, architecture and security notes live in [docs/design.md](docs/design.md).
> What is implemented and verified so far lives in [docs/implementation-report.md](docs/implementation-report.md).

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
- Go 1.25+ and Node 20+ for running the services outside containers (frontend in a later task)

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

Other targets: `make help`.

## Layout

```
backend/
  cmd/api/        entry point, wiring, graceful shutdown
  internal/
    config/       environment parsing and validation
    database/     pgx connection pool
    httpx/        JSON error envelope and typed application errors
    middleware/   request logging, panic recovery
    server/       router assembly
  migrations/     golang-migrate SQL migrations
docs/             design notes and AI-usage log
```

`docker-compose.yml` defines PostgreSQL and a one-shot `migrate` service. The frontend service is
added in a later task.
