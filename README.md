# booking-bot

Telegram-бот для онлайн-записи клиентов на услуги (барбершоп, салон красоты,
частный мастер и т.д.). Production-like pet-project на Go.

> Статус: Этап 1 — Infrastructure. Бизнес-логики и Telegram-интеграции пока
> нет; полное README появится на финальном этапе разработки.

## Стек

Go 1.24, PostgreSQL, Redis, Docker Compose, pgx, goose, slog.

## Запуск (Этап 1)

```bash
cp .env.example .env
# заполнить BOT_TOKEN не обязательно на этом этапе

docker compose up --build
```

Ожидаемый результат в логах контейнера `bot`:

```
starting booking-bot ...
connected to postgres
connected to redis
booking-bot is up, waiting for shutdown signal
```

Остановка: `Ctrl+C` или `docker compose down` — приложение должно завершиться
без ошибок (graceful shutdown по SIGINT/SIGTERM).

## Локальный запуск без Docker

```bash
make tidy   # первый раз — подтянуть зависимости и сгенерировать go.sum
docker compose up -d postgres redis
export $(cat .env.example | xargs)
make run
```

## Миграции (goose)

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
export DATABASE_URL=postgres://booking:booking@localhost:5432/booking?sslmode=disable
make migrate-up
```

На этом этапе единственная миграция включает расширение `btree_gist`,
которое понадобится в Этапе 6 для защиты от race condition при бронировании.

## Дальше — Этап 2

Users + Telegram: подключение Telegram Bot API, обработка `/start`,
`users` repository/service, главное меню.
