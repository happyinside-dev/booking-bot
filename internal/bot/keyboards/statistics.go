package keyboards

import "github.com/go-telegram/bot/models"

func StatisticsKeyboard() models.ReplyKeyboardMarkup {
	return reply([][]string{{"📊 Сегодня", "📊 7 дней"}, {"📊 30 дней", "📊 Всё время"}, {"🔙 Админ меню"}})
}
