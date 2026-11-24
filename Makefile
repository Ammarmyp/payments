# Load local secrets/config. Copy .env.example → .env if missing.
-include .env
export

SERVICE ?= wallet

# Wallet migrations use DATABASE_URL. Other services: make migrate-up SERVICE=auth DB_URL=...
ifeq ($(SERVICE),wallet)
DB_URL ?= $(DATABASE_URL)
else
DB_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:5432/$(SERVICE)?sslmode=disable
endif

MIGRATE := migrate -path db/$(SERVICE)/migrations -database "$(DB_URL)"

# Stamped into binaries via -ldflags (see run-wallet).
VERSION ?= $(shell git describe --always --dirty)

.PHONY: bootstrap up down migrate-up migrate-down migrate-version migrate-new sqlc test run-wallet

define require_env_file
$(if $(wildcard .env),,$(error missing .env — copy .env.example to .env))
endef

define require_database_url
$(call require_env_file)
$(if $(strip $(DATABASE_URL)),,$(error DATABASE_URL is not set in .env))
endef

define require_test_database_url
$(call require_env_file)
$(if $(strip $(TEST_DATABASE_URL)),,$(error TEST_DATABASE_URL is not set in .env))
endef

# One-shot reopen: infra + migrations.
bootstrap: up migrate-up
	@echo Ready. Run Wallet with: make run-wallet
	@echo Run tests with:      make test

up:
	docker compose up -d --wait

down:
	docker compose down

migrate-up:
	$(call require_database_url)
	$(if $(strip $(DB_URL)),,$(error DB_URL is empty — check DATABASE_URL / SERVICE))
	@$(MIGRATE) up

migrate-down:
	$(call require_database_url)
	@$(MIGRATE) down 1

migrate-version:
	$(call require_database_url)
	@$(MIGRATE) version

migrate-new:
	$(if $(strip $(name)),,$(error usage: make migrate-new name=add_something))
	migrate create -ext sql -dir db/$(SERVICE)/migrations -seq $(name)

# Regenerate Wallet sqlc code from db/wallet/migrations (*.up.sql) + queries.
# Keep zero-padded migrate -seq names so sqlc lexical order matches apply order.
sqlc:
	sqlc generate

test:
	$(call require_test_database_url)
	go test ./...

# Build then run Wallet against the local database, stamped with VERSION.
run-wallet:
	$(call require_database_url)
	@go build -ldflags "-X main.version=$(VERSION)" -o bin/wallet ./cmd/wallet
	@./bin/wallet
