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
# Note: the app also applies these migrations automatically on startup
# (see internal/database/migrate.go); these targets are for manual/local use.
migrate-up:
	goose -dir internal/database/migrations postgres "$$DATABASE_URL" up

migrate-down:
	goose -dir internal/database/migrations postgres "$$DATABASE_URL" down
