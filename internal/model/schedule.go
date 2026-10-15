package model

import "time"

// WorkingHours describes one local wall-clock interval in a business day.
// Minutes are counted from local midnight. Weekday follows ISO: Monday=1.
type WorkingHours struct {
	ID          int64
	EmployeeID  int64
	Weekday     int
	StartMinute int
	EndMinute   int
	IsDayOff    bool
}

type BlockedSlot struct {
	ID         int64
	EmployeeID int64
	StartAt    time.Time
	EndAt      time.Time
	Reason     string
	CreatedAt  time.Time
}

type TimeInterval struct {
	Start time.Time
	End   time.Time
}
