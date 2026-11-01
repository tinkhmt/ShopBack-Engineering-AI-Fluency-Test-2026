SHELL := /bin/bash
COMPOSE := docker compose
MIGRATE := $(COMPOSE) run --rm migrate

.PHONY: help db-up down db-reset migrate-up migrate-down migrate-version api test

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
	set -a && source .env && set +a && cd backend && go run ./cmd/api

test: ## run backend tests
	cd backend && go test ./...
