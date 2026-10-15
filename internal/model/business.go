package model

import "time"

type Business struct {
	ID                  int64
	Name                string
	Description         string
	Address             string
	Timezone            string
	SlotIntervalMin     int
	MinBookingNoticeMin int
	MaxBookingDays      int

	CreatedAt time.Time
	UpdatedAt time.Time
}
