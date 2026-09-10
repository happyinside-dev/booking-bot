// Package keyboards builds Telegram reply/inline keyboards. It contains no
// business logic -- only layout.
package keyboards

import "github.com/go-telegram/bot/models"

// MainMenu is the top-level client menu shown after /start.
func MainMenu() models.ReplyKeyboardMarkup {
	return models.ReplyKeyboardMarkup{
		Keyboard: [][]models.KeyboardButton{
			{{Text: "📅 Записаться"}, {Text: "📋 Мои записи"}},
			{{Text: "💈 Услуги"}, {Text: "ℹ️ О компании"}},
		},
		ResizeKeyboard: true,
	}
}
