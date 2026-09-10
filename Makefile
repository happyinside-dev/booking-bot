.PHONY: tidy build run up down logs migrate-up migrate-down

tidy:
	go mod tidy

build:
	go build -o bin/bot ./cmd/bot

run: build
	./bin/bot

up:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f bot

# Requires: go install github.com/pressly/goose/v3/cmd/goose@latest
migrate-up:
	goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down:
	goose -dir migrations postgres "$$DATABASE_URL" down
