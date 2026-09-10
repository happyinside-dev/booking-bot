// Package botapp wires up the Telegram bot instance: it creates the
// underlying client and registers routes against internal/bot/handlers.
// It does not itself contain business or presentation logic.
package botapp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-telegram/bot"

	"github.com/yourname/booking-bot/internal/bot/handlers"
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
