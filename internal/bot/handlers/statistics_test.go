package handlers

import (
	"strings"
	"testing"

	"github.com/happyinside-dev/booking-bot/internal/model"
)

func TestFormatStatistics(t *testing.T) {
	t.Parallel()
	message := formatStatistics(&model.BusinessStatistics{Total: 10, Upcoming: 2, Completed: 5, Cancelled: 2, NoShow: 1, CancellationRate: 20, ActualRevenue: 5000, PotentialRevenue: 7000, PopularServices: []model.ServiceStatistics{{Name: "Стрижка", Bookings: 4}}, EmployeeLoad: []model.EmployeeStatistics{{Name: "Иван", Bookings: 6, BookedMinutes: 300}}}, "30 дней")
	for _, want := range []string{"Всего записей: 10", "Процент отмен: 20.0%", "5000.00 ₽", "Стрижка — 4", "Иван — 6 записей, 300 мин."} {
		if !strings.Contains(message, want) {
			t.Errorf("message does not contain %q: %s", want, message)
		}
	}
}
func TestFormatEmptyStatistics(t *testing.T) {
	t.Parallel()
	message := formatStatistics(&model.BusinessStatistics{}, "сегодня")
	if strings.Count(message, "нет данных") != 2 {
		t.Fatalf("message=%s", message)
	}
}
