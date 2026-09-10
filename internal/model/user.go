package model

import "time"

// User represents a Telegram user known to the system. It is created (or
// refreshed) on every /start and is independent from business membership:
// a User only becomes staff once linked via a business_members row
// (introduced in Stage 3).
type User struct {
	ID         int64
	TelegramID int64
	Username   *string
	FirstName  *string
	LastName   *string
	Phone      *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
