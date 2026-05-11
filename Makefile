.PHONY: build run test lint tidy docker-build docker-up docker-down migrate-create migrate-up migrate-down
include .env
export

APP          := notification-service
BIN          := bin/$(APP)
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/notification_db?sslmode=disable
PROJECT_ROOT := $(CURDIR)
export PROJECT_ROOT


build:
	go build -ldflags="-s -w" -o $(BIN) ./cmd/server

run: build
	./$(BIN)

test:
	go test -v -race -count=1 ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

docker-build:
	docker build -t $(APP) .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

migrate-create:
ifndef name
	@echo Error: name is undefined.
	@echo Usage: make migrate-create name=init
	@exit 1
endif
	docker compose run --rm notification-migration create -ext sql -dir /migrations -seq $(name)

migrate-up:
	@make migrate-action action=up

migrate-down:
	@make migrate-action action=down

migrate-action:
	@docker compose run --rm notification-migration \
		-path=/migrations/ \
		-database "postgres://${DATABASE_USER}:${DATABASE_PASSWORD}@postgres:5432/${DATABASE_DBNAME}?sslmode=disable" \
		$(action)
