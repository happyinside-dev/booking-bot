package model

import "time"

type Business struct {
	ID          int64
	Name        string
	Description string
	Address     string
	Timezone    string

	CreatedAt time.Time
	UpdatedAt time.Time
}
