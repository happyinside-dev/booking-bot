package service

import (
	"fmt"
	"time"

	"github.com/happyinside-dev/booking-bot/internal/model"
)

type NotificationService struct{}

func NewNotificationService() *NotificationService { return &NotificationService{} }

func (s *NotificationService) Format(n model.Notification) (string, error) {
	loc, err := time.LoadLocation(n.BusinessTimezone)
	if err != nil {
		return "", fmt.Errorf("notification: invalid timezone %q: %w", n.BusinessTimezone, err)
	}
	start := n.BookingStart.In(loc)
	lead := "До записи остался час."
	if n.Type == model.NotificationReminder24H {
		lead = "До записи остались сутки."
	}
	return fmt.Sprintf("⏰ Напоминание о записи\n\nУслуга: %s\nМастер: %s\nДата: %s\nВремя: %s\nСтоимость: %.2f ₽\n\n%s", n.ServiceName, n.EmployeeName, start.Format("02.01.2006"), start.Format("15:04"), n.Price, lead), nil
}
