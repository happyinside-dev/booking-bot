package model

import "time"

type Service struct {
	ID          int64
	BusinessID  int64
	Name        string
	Description string

	DurationMin int
	Price       float64

	IsActive bool

	CreatedAt time.Time
	UpdatedAt time.Time
}
