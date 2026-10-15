package model

import "time"

type BusinessBot struct {
	ID         int64
	BusinessID int64
	Username   string
	TokenKey   string
	IsActive   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
