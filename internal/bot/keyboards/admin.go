package keyboards

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func AdminKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📅 Записи сегодня"),
			tgbotapi.NewKeyboardButton("📆 Все записи"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("💈 Услуги"),
			tgbotapi.NewKeyboardButton("👨‍💼 Сотрудники"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🕐 Расписание"),
			tgbotapi.NewKeyboardButton("🚫 Заблокировать время"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("👥 Клиенты"),
			tgbotapi.NewKeyboardButton("📊 Статистика"),
		),
	)
}
