ifneq (,$(wildcard ./.env))
    include .env
    export
endif

INITDB_PATH=deployments/docker/postgres/initdb.d
MIGRATION_URL="postgres://$(RPN_DB_USER):$(RPN_DB_PASSWORD)@localhost:5432/$(RPN_DB_DBNAME)?sslmode=false"

.PHONY: up down migrate-up run verify perms

verify-perms: ## Ensure init scripts are executable
	@chmod +x $(INITDB_PATH)/*.sh

up: ## Start infra and run migrations
	docker compose up -d

down: ## Stop everything
	docker compose down

run: up
	go run ./cmd/server/main.go