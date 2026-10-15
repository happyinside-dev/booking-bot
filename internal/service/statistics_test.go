package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/happyinside-dev/booking-bot/internal/apperr"
	"github.com/happyinside-dev/booking-bot/internal/model"
)

type statisticsRepoFake struct {
	stats    *model.BusinessStatistics
	from, to *time.Time
	now      time.Time
	called   bool
}

func (f *statisticsRepoFake) Get(_ context.Context, _ int64, from, to *time.Time, now time.Time) (*model.BusinessStatistics, error) {
	f.from, f.to, f.now, f.called = from, to, now, true
	copy := *f.stats
	return &copy, nil
}

func TestStatisticsServiceBoundsAndRate(t *testing.T) {
	t.Parallel()
	loc, _ := time.LoadLocation("Europe/Amsterdam")
	now := time.Date(2026, 3, 29, 12, 0, 0, 0, loc)
	tests := []struct {
		name             string
		period           StatisticsPeriod
		wantFrom, wantTo *time.Time
	}{
		{"today", StatisticsToday, timePtr(time.Date(2026, 3, 28, 23, 0, 0, 0, time.UTC)), timePtr(time.Date(2026, 3, 29, 22, 0, 0, 0, time.UTC))},
		{"7 days", Statistics7Days, timePtr(time.Date(2026, 3, 22, 23, 0, 0, 0, time.UTC)), timePtr(time.Date(2026, 3, 29, 22, 0, 0, 0, time.UTC))},
		{"30 days", Statistics30Days, timePtr(time.Date(2026, 2, 27, 23, 0, 0, 0, time.UTC)), timePtr(time.Date(2026, 3, 29, 22, 0, 0, 0, time.UTC))},
		{"all", StatisticsAll, nil, nil},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &statisticsRepoFake{stats: &model.BusinessStatistics{Total: 10, Cancelled: 2}}
			svc := NewStatisticsService(repo, bookingBusinessRepoFake{business: &model.Business{ID: 1, Timezone: "Europe/Amsterdam"}}, authorizerFake{})
			svc.now = func() time.Time { return now }
			stats, err := svc.Get(context.Background(), 5, 1, tt.period)
			if err != nil {
				t.Fatal(err)
			}
			if stats.CancellationRate != 20 {
				t.Fatalf("rate=%v", stats.CancellationRate)
			}
			assertOptionalTime(t, repo.from, tt.wantFrom)
			assertOptionalTime(t, repo.to, tt.wantTo)
		})
	}
}
func TestStatisticsCancellationRateWithNoBookings(t *testing.T) {
	t.Parallel()
	repo := &statisticsRepoFake{stats: &model.BusinessStatistics{}}
	svc := NewStatisticsService(repo, bookingBusinessRepoFake{business: &model.Business{ID: 1, Timezone: "UTC"}}, authorizerFake{})
	stats, err := svc.Get(context.Background(), 5, 1, StatisticsAll)
	if err != nil {
		t.Fatal(err)
	}
	if stats.CancellationRate != 0 {
		t.Fatalf("rate=%v", stats.CancellationRate)
	}
}
func TestStatisticsServiceAuthorization(t *testing.T) {
	t.Parallel()
	repo := &statisticsRepoFake{stats: &model.BusinessStatistics{}}
	svc := NewStatisticsService(repo, bookingBusinessRepoFake{business: &model.Business{Timezone: "UTC"}}, authorizerFake{err: apperr.ErrForbidden})
	_, err := svc.Get(context.Background(), 1, 1, StatisticsToday)
	if !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("error=%v", err)
	}
	if repo.called {
		t.Fatal("repository called for forbidden user")
	}
}
func TestStatisticsInvalidPeriod(t *testing.T) {
	t.Parallel()
	_, _, err := statisticsBounds("year", time.Now())
	if !errors.Is(err, apperr.ErrInvalidInput) {
		t.Fatalf("error=%v", err)
	}
}
func timePtr(value time.Time) *time.Time { return &value }
func assertOptionalTime(t *testing.T, got, want *time.Time) {
	t.Helper()
	if got == nil || want == nil {
		if got != want {
			t.Fatalf("got=%v want=%v", got, want)
		}
		return
	}
	if !got.Equal(*want) {
		t.Fatalf("got=%v want=%v", got, want)
	}
}
