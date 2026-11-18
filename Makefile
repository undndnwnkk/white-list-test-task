-include .env
export

MIGRATIONS_DIR ?= migrations

.PHONY: up down run build db-up db-down migrate-create migrate-up migrate-down migrate-reset migrate-status test

up:
	docker compose up --build -d --wait

down:
	docker compose down

run:
	go run ./cmd/app

build:
	go build -o bin/app ./cmd/app

db-up:
	docker compose up -d --wait postgres

db-down:
	docker compose down

migrate-create:
	goose -dir $(MIGRATIONS_DIR) -s create $(name) sql

migrate-up:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_DSN)" up

migrate-down:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_DSN)" down

migrate-reset:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_DSN)" reset

migrate-status:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_DSN)" status

test: export TEST_DB_DSN = $(DB_DSN)
test:
	go test -count=1 -p 1 ./...
