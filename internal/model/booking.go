package model

import "time"

type BookingStatus string

const (
	BookingPending   BookingStatus = "PENDING"
	BookingConfirmed BookingStatus = "CONFIRMED"
	BookingCancelled BookingStatus = "CANCELLED"
	BookingCompleted BookingStatus = "COMPLETED"
	BookingNoShow    BookingStatus = "NO_SHOW"
)

type Booking struct {
	ID, BusinessID, EmployeeID, ServiceID, UserID int64
	StartAt, EndAt                                time.Time
	Status                                        BookingStatus
	Price                                         float64
	Comment                                       string
	CreatedAt, UpdatedAt                          time.Time
}

type BookingView struct {
	Booking
	ServiceName, EmployeeName, BusinessName, BusinessTimezone string
}
