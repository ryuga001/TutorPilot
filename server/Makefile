.PHONY: help setup tidy run run-worker build test swagger migrate-up migrate-down migrate-version \
        infra-up infra-down infra-logs new-migration dlq

ifneq (,$(wildcard .env))
	include .env
	export
endif

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

setup:
	@test -f .env || cp .env.example .env
	go mod download

tidy:
	go mod tidy

run:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

test:
	go test ./...

swagger:
	go run github.com/swaggo/swag/cmd/swag init -g cmd/api/main.go -o docs --parseInternal

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-version:
	go run ./cmd/migrate version

new-migration:
	@test -n "$(name)" || (echo "usage: make new-migration name=<name>"; exit 1)
	@ts=$$(date +%Y%m%d%H%M%S); \
	touch migrations/$${ts}_$(name).up.sql migrations/$${ts}_$(name).down.sql; \
	echo "created migrations/$${ts}_$(name).{up,down}.sql"


dlq:
	@psql "$(DATABASE_URL)" -c "SELECT event_id, event_type, attempts, failed_at, last_error FROM dead_events WHERE replayed_at IS NULL ORDER BY failed_at DESC LIMIT 50;"

infra-up:
	docker compose up -d

infra-down:
	docker compose down

infra-logs:
	docker compose logs -f
