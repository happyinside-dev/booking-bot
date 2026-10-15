package service

import (
	"strings"
	"testing"
	"time"

	"github.com/happyinside-dev/booking-bot/internal/model"
)

func TestNotificationFormatUsesBusinessTimezone(t *testing.T) {
	t.Parallel()
	service := NewNotificationService()
	message, err := service.Format(model.Notification{Type: model.NotificationReminder24H, BookingStart: time.Date(2026, 10, 16, 11, 30, 0, 0, time.UTC), BusinessTimezone: "Europe/Moscow", ServiceName: "Стрижка", EmployeeName: "Иван", Price: 1500})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"16.10.2026", "14:30", "Стрижка", "Иван", "1500.00", "сутки"} {
		if !strings.Contains(message, want) {
			t.Errorf("message %q does not contain %q", message, want)
		}
	}
}

func TestNotificationFormatRejectsTimezone(t *testing.T) {
	t.Parallel()
	_, err := NewNotificationService().Format(model.Notification{BusinessTimezone: "Mars/Olympus"})
	if err == nil {
		t.Fatal("expected timezone error")
	}
}
