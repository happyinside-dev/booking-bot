package keyboards

import "github.com/go-telegram/bot/models"

func AdminKeyboard() models.ReplyKeyboardMarkup {
	return reply([][]string{
		{"📅 Записи сегодня", "📆 Все записи"},
		{"🛠 Услуги", "👨‍💼 Сотрудники"},
		{"🕐 Расписание", "🚫 Заблокировать время"},
		{"📊 Статистика", "🔙 Главное меню"},
	})
}

func ServiceAdminKeyboard() models.ReplyKeyboardMarkup {
	return reply([][]string{{"➕ Добавить услугу"}, {"✏️ Изменить услугу"}, {"🔄 Статус услуги"}, {"🔙 Админ меню"}})
}

func EmployeeAdminKeyboard() models.ReplyKeyboardMarkup {
	return reply([][]string{{"➕ Добавить сотрудника"}, {"✏️ Изменить сотрудника"}, {"🔄 Статус сотрудника"}, {"🔙 Админ меню"}})
}

func CancelKeyboard() models.ReplyKeyboardMarkup { return reply([][]string{{"❌ Отмена"}}) }

func reply(rows [][]string) models.ReplyKeyboardMarkup {
	keyboard := make([][]models.KeyboardButton, len(rows))
	for i, row := range rows {
		keyboard[i] = make([]models.KeyboardButton, len(row))
		for j, label := range row {
			keyboard[i][j] = models.KeyboardButton{Text: label}
		}
	}
	return models.ReplyKeyboardMarkup{Keyboard: keyboard, ResizeKeyboard: true}
}
