-include .env
export

DB_PORT     ?= 5433
DB_DSN      ?= postgres://crm:crm@localhost:$(DB_PORT)/crm?sslmode=disable
TEST_DB_DSN ?= postgres://crm:crm@localhost:$(DB_PORT)/crm_test?sslmode=disable

.PHONY: up down logs run build db-up test test-docker smoke migrate-create

## Run the whole stack (PostgreSQL + API) in Docker.
up:
	docker compose up --build -d --wait

down:
	docker compose down

logs:
	docker compose logs -f app

## Run tests inside Docker: needs only Docker, no local Go.
test-docker:
	docker compose run --rm --build tests

## Run tests with local Go against PostgreSQL from docker compose.
test: db-up
	go test -count=1 ./...

## Run the API with local Go against PostgreSQL from docker compose.
run: db-up
	go run ./cmd/app

build:
	go build -o bin/app ./cmd/app

db-up:
	docker compose up -d --wait postgres

## End-to-end scenario with curl against a running API.
smoke:
	./scripts/smoke.sh

## Create a new SQL migration (needs the goose CLI). Migrations are applied on app start.
migrate-create:
	goose -dir migrations -s create $(name) sql
