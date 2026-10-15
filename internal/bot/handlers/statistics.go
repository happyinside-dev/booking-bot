package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/happyinside-dev/booking-bot/internal/bot/keyboards"
	"github.com/happyinside-dev/booking-bot/internal/model"
	"github.com/happyinside-dev/booking-bot/internal/service"
)

func (h *Handler) StatisticsMenu(ctx context.Context, b *bot.Bot, u *models.Update) {
	if _, ok := h.requireAdmin(ctx, b, u); !ok {
		return
	}
	h.sendWithKeyboard(ctx, b, u.Message.Chat.ID, "Выберите период статистики:", keyboards.StatisticsKeyboard())
}
func (h *Handler) StatisticsToday(ctx context.Context, b *bot.Bot, u *models.Update) {
	h.showStatistics(ctx, b, u, service.StatisticsToday, "сегодня")
}
func (h *Handler) Statistics7Days(ctx context.Context, b *bot.Bot, u *models.Update) {
	h.showStatistics(ctx, b, u, service.Statistics7Days, "7 дней")
}
func (h *Handler) Statistics30Days(ctx context.Context, b *bot.Bot, u *models.Update) {
	h.showStatistics(ctx, b, u, service.Statistics30Days, "30 дней")
}
func (h *Handler) StatisticsAll(ctx context.Context, b *bot.Bot, u *models.Update) {
	h.showStatistics(ctx, b, u, service.StatisticsAll, "всё время")
}
func (h *Handler) showStatistics(ctx context.Context, b *bot.Bot, u *models.Update, period service.StatisticsPeriod, label string) {
	admin, ok := h.requireAdmin(ctx, b, u)
	if !ok {
		return
	}
	stats, err := h.statisticsService.Get(ctx, admin.user.ID, admin.business.ID, period)
	if err != nil {
		h.adminError(ctx, b, u.Message.Chat.ID, err)
		return
	}
	h.sendWithKeyboard(ctx, b, u.Message.Chat.ID, formatStatistics(stats, label), keyboards.StatisticsKeyboard())
}

func formatStatistics(stats *model.BusinessStatistics, period string) string {
	var out strings.Builder
	fmt.Fprintf(&out, "📊 Статистика: %s\n\nВсего записей: %d\nБудущие: %d\nЗавершённые: %d\nОтменённые: %d\nНе пришли: %d\nПроцент отмен: %.1f%%\n\nФактическая выручка: %.2f ₽\nПотенциальная выручка: %.2f ₽", period, stats.Total, stats.Upcoming, stats.Completed, stats.Cancelled, stats.NoShow, stats.CancellationRate, stats.ActualRevenue, stats.PotentialRevenue)
	out.WriteString("\n\nПопулярные услуги:")
	if len(stats.PopularServices) == 0 {
		out.WriteString(" нет данных")
	} else {
		for i, item := range stats.PopularServices {
			fmt.Fprintf(&out, "\n%d. %s — %d", i+1, item.Name, item.Bookings)
		}
	}
	out.WriteString("\n\nЗагрузка сотрудников:")
	if len(stats.EmployeeLoad) == 0 {
		out.WriteString(" нет данных")
	} else {
		for _, item := range stats.EmployeeLoad {
			fmt.Fprintf(&out, "\n%s — %d записей, %d мин.", item.Name, item.Bookings, item.BookedMinutes)
		}
	}
	return out.String()
}
