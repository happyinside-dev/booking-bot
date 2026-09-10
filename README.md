# booking-bot

Telegram-бот для онлайн-записи клиентов на услуги (барбершоп, салон красоты,
частный мастер и т.д.). Production-like pet-project на Go.

> Статус: Этап 2 — Users + Telegram. Реализован `/start` с регистрацией
> пользователя и главное меню; RBAC, бизнесы и остальная бизнес-логика — в
> следующих этапах. Полное README появится на финальном этапе разработки.

## Стек

Go 1.24, PostgreSQL, Redis, Docker Compose, pgx, goose, slog,
[go-telegram/bot](https://github.com/go-telegram/bot).

## Запуск (Этап 2)

```bash
cp .env.example .env
# заполнить BOT_TOKEN значением от @BotFather — теперь обязательно

docker compose up --build
```

Ожидаемый результат в логах контейнера `bot`:

```
starting booking-bot ...
migrations applied
connected to postgres
connected to redis
telegram bot started (long polling)
```

Отправьте боту `/start` в Telegram — он должен ответить "Добро пожаловать!"
с клавиатурой из четырёх кнопок. Нажатие любой кнопки пока отвечает
"эта функция в разработке" — это ожидаемо на этом этапе.

Проверка регистрации: в базе должна появиться строка в `users` с вашим
`telegram_id`.

```bash
docker compose exec postgres psql -U booking -d booking -c "SELECT id, telegram_id, username FROM users;"
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

Приложение **автоматически применяет все миграции при старте**
(`internal/database/migrate.go`, файлы встроены в бинарник через
`embed.FS` из `internal/database/migrations`) — отдельный шаг не нужен ни
локально, ни в Docker.

Для ручного управления (например, чтобы применить/откатить миграции без
запуска бота) можно использовать CLI:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
export DATABASE_URL=postgres://booking:booking@localhost:5432/booking?sslmode=disable
make migrate-up
make migrate-down
```

На этом этапе миграции: `btree_gist` (понадобится в Этапе 6 для защиты от
race condition при бронировании) и таблица `users`.

> Осознанный компромисс: автоприменение на каждом старте — нормально для
> одного инстанса бота. При нескольких репликах в проде миграции обычно
> запускают отдельным job/init-шагом, а не из каждого процесса параллельно.

## Дальше — Этап 3

Businesses + RBAC: таблицы `businesses`/`business_members`, роли
OWNER/ADMIN/EMPLOYEE, централизованная проверка доступа, команда `/admin`.
