// Package handlers translates Telegram updates into service calls and
// service results into Telegram messages. It must never contain business
// rules -- those live in internal/service.
package handlers

import (
	"context"
	"errors"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/happyinside-dev/booking-bot/internal/apperr"
	"github.com/happyinside-dev/booking-bot/internal/bot/keyboards"
	"github.com/happyinside-dev/booking-bot/internal/model"
	"github.com/happyinside-dev/booking-bot/internal/service"
)

type Handler struct {
	userService     *service.UserService
	businessService *service.BusinessService
	log             *slog.Logger
}

func New(
	userService *service.UserService,
	businessService *service.BusinessService,
	log *slog.Logger,
) *Handler {

	return &Handler{
		userService:     userService,
		businessService: businessService,
		log:             log,
	}
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

func (h *Handler) Admin(ctx context.Context, b *bot.Bot, update *models.Update) {

	if update.Message == nil {
		return
	}

	user, err := h.userService.GetByTelegramID(ctx, update.Message.From.ID)
	if err != nil {
		h.sendText(ctx, b, update.Message.Chat.ID, "❌ Пользователь не найден.")
		return
	}

	businessID, err := h.businessService.FirstBusiness(ctx, user.ID)
	if err != nil {
		h.sendText(ctx, b, update.Message.Chat.ID, "❌ У вас нет бизнеса.")
		return
	}

	err = h.businessService.HasRole(
		ctx,
		businessID,
		user.ID,
		model.RoleOwner,
		model.RoleAdmin,
	)

	if err != nil {
		if errors.Is(err, apperr.ErrForbidden) {
			h.sendText(ctx, b, update.Message.Chat.ID, "❌ У вас нет доступа к административному меню.")
			return
		}

		h.log.Error("failed to check role", slog.Any("error", err))
		h.sendText(ctx, b, update.Message.Chat.ID, "Произошла ошибка.")
		return
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "🔧 Панель администратора",
		ReplyMarkup: keyboards.AdminKeyboard(),
	})

	if err != nil {
		h.log.Error("failed to send admin menu", slog.Any("error", err))
	}
}
