package service

import (
	"testing"
	"time"

	"github.com/happyinside-dev/booking-bot/internal/model"
)

func TestComputeAvailableSlots(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, time.September, 13, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		duration  time.Duration
		interval  time.Duration
		minStart  time.Time
		busy      []model.TimeInterval
		wantHours []string
	}{
		{
			name:     "30 minute service",
			duration: 30 * time.Minute, interval: 30 * time.Minute, minStart: base,
			wantHours: []string{"10:00", "10:30", "11:00", "11:30", "12:00", "12:30", "13:00", "13:30"},
		},
		{
			name:     "duration cannot exceed work end",
			duration: 90 * time.Minute, interval: 30 * time.Minute, minStart: base,
			wantHours: []string{"10:00", "10:30", "11:00", "11:30", "12:00", "12:30"},
		},
		{
			name:     "blocked interval and adjacent slots",
			duration: 30 * time.Minute, interval: 30 * time.Minute, minStart: base,
			busy:      []model.TimeInterval{{Start: base.Add(time.Hour), End: base.Add(2 * time.Hour)}},
			wantHours: []string{"10:00", "10:30", "12:00", "12:30", "13:00", "13:30"},
		},
		{
			name:     "long service does not fit into gap",
			duration: 60 * time.Minute, interval: 30 * time.Minute, minStart: base,
			busy: []model.TimeInterval{
				{Start: base, End: base.Add(time.Hour)},
				{Start: base.Add(150 * time.Minute), End: base.Add(195 * time.Minute)},
			},
			wantHours: []string{"11:00", "11:30"},
		},
		{
			name:     "minimum booking notice",
			duration: 30 * time.Minute, interval: 30 * time.Minute, minStart: base.Add(75 * time.Minute),
			wantHours: []string{"11:30", "12:00", "12:30", "13:00", "13:30"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ComputeAvailableSlots(base, base.Add(4*time.Hour), tt.duration, tt.interval, tt.minStart, tt.busy)
			if len(got) != len(tt.wantHours) {
				t.Fatalf("got %d slots (%v), want %d", len(got), got, len(tt.wantHours))
			}
			for i, slot := range got {
				if value := slot.Format("15:04"); value != tt.wantHours[i] {
					t.Errorf("slot %d = %s, want %s", i, value, tt.wantHours[i])
				}
			}
		})
	}
}

func TestComputeAvailableSlotsDST(t *testing.T) {
	t.Parallel()
	loc, err := time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, time.March, 29, 1, 0, 0, 0, loc)
	end := time.Date(2026, time.March, 29, 5, 0, 0, 0, loc)
	got := ComputeAvailableSlots(start, end, time.Hour, time.Hour, start, nil)
	if len(got) != 3 {
		t.Fatalf("got %d slots, want 3: %v", len(got), got)
	}
	wantLocal := []string{"01:00", "03:00", "04:00"}
	for i := range got {
		if value := got[i].In(loc).Format("15:04"); value != wantLocal[i] {
			t.Errorf("slot %d = %s, want %s", i, value, wantLocal[i])
		}
	}
}
