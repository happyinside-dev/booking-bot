package model

import "time"

type Employee struct {
	ID         int64
	BusinessID int64

	UserID *int64

	Name        string
	Description string

	IsActive bool

	CreatedAt time.Time
	UpdatedAt time.Time
}
