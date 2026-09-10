// Package handlers translates Telegram updates into service calls and
// service results into Telegram messages. It must never contain business
// rules -- those live in internal/service.
package handlers

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/yourname/booking-bot/internal/bot/keyboards"
	"github.com/yourname/booking-bot/internal/service"
)

type Handler struct {
	userService *service.UserService
	log         *slog.Logger
}

func New(userService *service.UserService, log *slog.Logger) *Handler {
	return &Handler{userService: userService, log: log}
}

// Start handles the /start command: registers or refreshes the user, then
// shows the main menu.
func (h *Handler) Start(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}
	from := update.Message.From
	chatID := update.Message.Chat.ID

	_, err := h.userService.RegisterOrUpdate(ctx, service.RegisterUserInput{
		TelegramID: from.ID,
		Username:   from.Username,
		FirstName:  from.FirstName,
		LastName:   from.LastName,
	})
	if err != nil {
		h.log.Error("failed to register user on /start", slog.Any("error", err), slog.Int64("telegram_id", from.ID))
		h.sendText(ctx, b, chatID, "Произошла ошибка, попробуйте ещё раз чуть позже.")
		return
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        "Добро пожаловать!",
		ReplyMarkup: keyboards.MainMenu(),
	})
	if err != nil {
		h.log.Error("failed to send welcome message", slog.Any("error", err))
	}
}

// Default handles any update not matched by a more specific handler
// (including the not-yet-implemented main menu buttons).
func (h *Handler) Default(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	h.sendText(ctx, b, update.Message.Chat.ID, "❗ Эта функция пока в разработке.")
}

func (h *Handler) sendText(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text}); err != nil {
		h.log.Error("failed to send message", slog.Any("error", err), slog.Int64("chat_id", chatID))
	}
}
