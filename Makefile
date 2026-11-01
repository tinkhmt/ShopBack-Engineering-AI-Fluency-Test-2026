SHELL := /bin/bash
COMPOSE := docker compose
MIGRATE := $(COMPOSE) run --rm migrate
TEST_DATABASE_URL ?= postgres://todo:todo_dev_password@localhost:5432/todo_test?sslmode=disable

.PHONY: help db-up down db-reset migrate-up migrate-down migrate-version api test test-integration web

help: ## list available targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-16s %s\n", $$1, $$2}'

db-up: ## start postgres and wait until it is healthy
	$(COMPOSE) up -d --wait db

down: ## stop containers, keep data
	$(COMPOSE) down

db-reset: ## destroy the database volume, recreate it and migrate
	$(COMPOSE) down -v
	$(COMPOSE) up -d --wait db
	$(MIGRATE) up

migrate-up: ## apply all pending migrations
	$(MIGRATE) up

migrate-down: ## roll back the last migration
	$(MIGRATE) down 1

migrate-version: ## print the current schema version
	$(MIGRATE) version

api: ## run the API on the host against the local database
	set -a && source .env && set +a && cd backend && go run ./cmd/server

test: ## run backend tests
	cd backend && go test ./...

test-integration: db-up ## run the API integration suite against a dedicated test database
	@$(COMPOSE) exec -T db psql -U $${POSTGRES_USER:-todo} -d postgres -tAc \
		"SELECT 1 FROM pg_database WHERE datname = 'todo_test'" | grep -q 1 || \
		$(COMPOSE) exec -T db createdb -U $${POSTGRES_USER:-todo} todo_test
	cd backend && TEST_DATABASE_URL='$(TEST_DATABASE_URL)' go test -count=1 -v ./internal/integration/...

web: ## run the frontend dev server on :5173 (proxies /api to :8080)
	cd frontend && npm install --no-audit --no-fund && npm run dev
