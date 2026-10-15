package service

import (
	"context"
	"fmt"
	"time"

	"github.com/happyinside-dev/booking-bot/internal/apperr"
	"github.com/happyinside-dev/booking-bot/internal/model"
)

type StatisticsPeriod string

const (
	StatisticsToday  StatisticsPeriod = "today"
	Statistics7Days  StatisticsPeriod = "7_days"
	Statistics30Days StatisticsPeriod = "30_days"
	StatisticsAll    StatisticsPeriod = "all"
)

type statisticsRepository interface {
	Get(context.Context, int64, *time.Time, *time.Time, time.Time) (*model.BusinessStatistics, error)
}
type StatisticsService struct {
	repo       statisticsRepository
	businesses scheduleBusinessRepository
	auth       RoleAuthorizer
	now        func() time.Time
}

func NewStatisticsService(repo statisticsRepository, businesses scheduleBusinessRepository, auth RoleAuthorizer) *StatisticsService {
	return &StatisticsService{repo: repo, businesses: businesses, auth: auth, now: time.Now}
}

func (s *StatisticsService) Get(ctx context.Context, actorUserID, businessID int64, period StatisticsPeriod) (*model.BusinessStatistics, error) {
	if err := authorizeAdmin(ctx, s.auth, businessID, actorUserID); err != nil {
		return nil, err
	}
	business, err := s.businesses.GetByID(ctx, businessID)
	if err != nil {
		return nil, fmt.Errorf("service: statistics business: %w", err)
	}
	loc, err := time.LoadLocation(business.Timezone)
	if err != nil {
		return nil, apperr.ErrInvalidTimezone
	}
	now := s.now()
	from, to, err := statisticsBounds(period, now.In(loc))
	if err != nil {
		return nil, err
	}
	stats, err := s.repo.Get(ctx, businessID, from, to, now.UTC())
	if err != nil {
		return nil, fmt.Errorf("service: get statistics: %w", err)
	}
	if stats.Total > 0 {
		stats.CancellationRate = float64(stats.Cancelled) / float64(stats.Total) * 100
	}
	return stats, nil
}

func statisticsBounds(period StatisticsPeriod, now time.Time) (*time.Time, *time.Time, error) {
	if period == StatisticsAll {
		return nil, nil, nil
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := today.AddDate(0, 0, 1)
	start := today
	switch period {
	case StatisticsToday:
	case Statistics7Days:
		start = today.AddDate(0, 0, -6)
	case Statistics30Days:
		start = today.AddDate(0, 0, -29)
	default:
		return nil, nil, apperr.ErrInvalidInput
	}
	fromUTC, toUTC := start.UTC(), end.UTC()
	return &fromUTC, &toUTC, nil
}
