package keyboards

import "github.com/go-telegram/bot/models"

func BookingConfirmKeyboard() models.ReplyKeyboardMarkup {
	return reply([][]string{{"✅ Подтвердить запись"}, {"❌ Отмена"}})
}
func MyBookingsKeyboard() models.ReplyKeyboardMarkup {
	return reply([][]string{{"❌ Отменить запись"}, {"🔙 Главное меню"}})
}
