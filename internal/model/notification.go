package model

import "time"

type NotificationType string
type NotificationStatus string

const (
	NotificationReminder24H NotificationType = "REMINDER_24H"
	NotificationReminder1H  NotificationType = "REMINDER_1H"

	NotificationPending    NotificationStatus = "PENDING"
	NotificationProcessing NotificationStatus = "PROCESSING"
	NotificationSent       NotificationStatus = "SENT"
	NotificationFailed     NotificationStatus = "FAILED"
	NotificationCancelled  NotificationStatus = "CANCELLED"
)

type Notification struct {
	ID, BookingID, UserID, BusinessID int64
	Type                              NotificationType
	ScheduledAt                       time.Time
	SentAt                            *time.Time
	Status                            NotificationStatus
	Attempts                          int
	LockedUntil                       *time.Time
	LastError                         *string
	CreatedAt, UpdatedAt              time.Time

	TelegramID                                  int64
	BookingStart                                time.Time
	Price                                       float64
	ServiceName, EmployeeName, BusinessTimezone string
}
