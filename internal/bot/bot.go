// Package botapp wires up the Telegram bot instance: it creates the
// underlying client and registers routes against internal/bot/handlers.
// It does not itself contain business or presentation logic.
package botapp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-telegram/bot"

	"github.com/happyinside-dev/booking-bot/internal/bot/handlers"
)

type Bot struct {
	api *bot.Bot
	log *slog.Logger
}

func New(token string, h *handlers.Handler, log *slog.Logger) (*Bot, error) {
	app := &Bot{log: log}

	api, err := bot.New(token, bot.WithDefaultHandler(h.Default))
	if err != nil {
		return nil, fmt.Errorf("botapp: failed to create bot: %w", err)
	}

	api.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, h.Start)
	api.RegisterHandler(bot.HandlerTypeMessageText, "/admin", bot.MatchTypeExact, h.Admin)
	api.RegisterHandler(bot.HandlerTypeMessageText, "💈 Услуги", bot.MatchTypeExact, h.ServicesMenu)
	api.RegisterHandler(bot.HandlerTypeMessageText, "👨‍💼 Сотрудники", bot.MatchTypeExact, h.EmployeesMenu)
	api.RegisterHandler(bot.HandlerTypeMessageText, "➕ Добавить услугу", bot.MatchTypeExact, h.BeginServiceCreate)
	api.RegisterHandler(bot.HandlerTypeMessageText, "✏️ Изменить услугу", bot.MatchTypeExact, h.BeginServiceUpdate)
	api.RegisterHandler(bot.HandlerTypeMessageText, "🔄 Статус услуги", bot.MatchTypeExact, h.BeginServiceStatus)
	api.RegisterHandler(bot.HandlerTypeMessageText, "➕ Добавить сотрудника", bot.MatchTypeExact, h.BeginEmployeeCreate)
	api.RegisterHandler(bot.HandlerTypeMessageText, "✏️ Изменить сотрудника", bot.MatchTypeExact, h.BeginEmployeeUpdate)
	api.RegisterHandler(bot.HandlerTypeMessageText, "🔄 Статус сотрудника", bot.MatchTypeExact, h.BeginEmployeeStatus)
	api.RegisterHandler(bot.HandlerTypeMessageText, "🕐 Расписание", bot.MatchTypeExact, h.BeginSchedule)
	api.RegisterHandler(bot.HandlerTypeMessageText, "🚫 Заблокировать время", bot.MatchTypeExact, h.BeginBlocked)
	api.RegisterHandler(bot.HandlerTypeMessageText, "🔙 Админ меню", bot.MatchTypeExact, h.BackToAdmin)
	api.RegisterHandler(bot.HandlerTypeMessageText, "❌ Отмена", bot.MatchTypeExact, h.Cancel)
	api.RegisterHandler(bot.HandlerTypeMessageText, "📅 Записаться", bot.MatchTypeExact, h.BeginBooking)
	api.RegisterHandler(bot.HandlerTypeMessageText, "📋 Мои записи", bot.MatchTypeExact, h.MyBookings)
	api.RegisterHandler(bot.HandlerTypeMessageText, "❌ Отменить запись", bot.MatchTypeExact, h.BeginBookingCancel)
	api.RegisterHandler(bot.HandlerTypeMessageText, "🔙 Главное меню", bot.MatchTypeExact, h.MainMenu)
	api.RegisterHandler(bot.HandlerTypeMessageText, "📅 Записи сегодня", bot.MatchTypeExact, h.AdminToday)
	api.RegisterHandler(bot.HandlerTypeMessageText, "📆 Все записи", bot.MatchTypeExact, h.AdminAll)
	api.RegisterHandler(bot.HandlerTypeMessageText, "📊 Статистика", bot.MatchTypeExact, h.StatisticsMenu)
	api.RegisterHandler(bot.HandlerTypeMessageText, "📊 Сегодня", bot.MatchTypeExact, h.StatisticsToday)
	api.RegisterHandler(bot.HandlerTypeMessageText, "📊 7 дней", bot.MatchTypeExact, h.Statistics7Days)
	api.RegisterHandler(bot.HandlerTypeMessageText, "📊 30 дней", bot.MatchTypeExact, h.Statistics30Days)
	api.RegisterHandler(bot.HandlerTypeMessageText, "📊 Всё время", bot.MatchTypeExact, h.StatisticsAll)

	app.api = api
	return app, nil
}

// Start begins long polling. It blocks until ctx is cancelled, at which
// point it returns so the caller's deferred cleanup (closing DB/Redis) can
// run -- this is the graceful shutdown path wired up in Stage 1.
func (b *Bot) Start(ctx context.Context) {
	b.log.Info("telegram bot started (long polling)")
	b.api.Start(ctx)
	b.log.Info("telegram bot stopped")
}

func (b *Bot) SendText(ctx context.Context, chatID int64, text string) error {
	_, err := b.api.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text})
	if err != nil {
		return fmt.Errorf("telegram: send message: %w", err)
	}
	return nil
}
