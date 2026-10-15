package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/happyinside-dev/booking-bot/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StatisticsRepository struct{ pool *pgxpool.Pool }

func NewStatisticsRepository(pool *pgxpool.Pool) *StatisticsRepository {
	return &StatisticsRepository{pool: pool}
}

func (r *StatisticsRepository) Get(ctx context.Context, businessID int64, from, to *time.Time, now time.Time) (*model.BusinessStatistics, error) {
	const filter = `business_id=$1 AND ($2::timestamptz IS NULL OR start_at >= $2) AND ($3::timestamptz IS NULL OR start_at < $3)`
	q := `SELECT COUNT(*),COUNT(*) FILTER(WHERE status IN ('PENDING','CONFIRMED') AND start_at >= $4),COUNT(*) FILTER(WHERE status='COMPLETED'),COUNT(*) FILTER(WHERE status='CANCELLED'),COUNT(*) FILTER(WHERE status='NO_SHOW'),COALESCE(SUM(price) FILTER(WHERE status IN ('PENDING','CONFIRMED','COMPLETED')),0),COALESCE(SUM(price) FILTER(WHERE status='COMPLETED'),0) FROM bookings WHERE ` + filter
	stats := &model.BusinessStatistics{}
	if err := r.pool.QueryRow(ctx, q, businessID, from, to, now).Scan(&stats.Total, &stats.Upcoming, &stats.Completed, &stats.Cancelled, &stats.NoShow, &stats.PotentialRevenue, &stats.ActualRevenue); err != nil {
		return nil, fmt.Errorf("repository: aggregate statistics: %w", err)
	}
	services, err := r.popularServices(ctx, businessID, from, to)
	if err != nil {
		return nil, err
	}
	stats.PopularServices = services
	employees, err := r.employeeLoad(ctx, businessID, from, to)
	if err != nil {
		return nil, err
	}
	stats.EmployeeLoad = employees
	return stats, nil
}

func (r *StatisticsRepository) popularServices(ctx context.Context, businessID int64, from, to *time.Time) ([]model.ServiceStatistics, error) {
	const q = `SELECT s.id,s.name,COUNT(*),COALESCE(SUM(b.price),0) FROM bookings b JOIN services s ON s.id=b.service_id
		WHERE b.business_id=$1 AND ($2::timestamptz IS NULL OR b.start_at >= $2) AND ($3::timestamptz IS NULL OR b.start_at < $3)
		AND b.status IN ('PENDING','CONFIRMED','COMPLETED') GROUP BY s.id,s.name ORDER BY COUNT(*) DESC,s.name LIMIT 5`
	rows, err := r.pool.Query(ctx, q, businessID, from, to)
	if err != nil {
		return nil, fmt.Errorf("repository: popular services: %w", err)
	}
	defer rows.Close()
	var out []model.ServiceStatistics
	for rows.Next() {
		var item model.ServiceStatistics
		if err := rows.Scan(&item.ServiceID, &item.Name, &item.Bookings, &item.Revenue); err != nil {
			return nil, fmt.Errorf("repository: scan service statistics: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate service statistics: %w", err)
	}
	return out, nil
}

func (r *StatisticsRepository) employeeLoad(ctx context.Context, businessID int64, from, to *time.Time) ([]model.EmployeeStatistics, error) {
	const q = `SELECT e.id,e.name,COUNT(*),COUNT(*) FILTER(WHERE b.status='CANCELLED'),
		COALESCE(SUM((EXTRACT(EPOCH FROM (b.end_at-b.start_at))/60)::bigint) FILTER(WHERE b.status IN ('PENDING','CONFIRMED','COMPLETED')),0)::bigint,
		COALESCE(SUM(b.price) FILTER(WHERE b.status IN ('PENDING','CONFIRMED','COMPLETED')),0)
		FROM bookings b JOIN employees e ON e.id=b.employee_id WHERE b.business_id=$1
		AND ($2::timestamptz IS NULL OR b.start_at >= $2) AND ($3::timestamptz IS NULL OR b.start_at < $3)
		GROUP BY e.id,e.name ORDER BY COUNT(*) DESC,e.name LIMIT 10`
	rows, err := r.pool.Query(ctx, q, businessID, from, to)
	if err != nil {
		return nil, fmt.Errorf("repository: employee load: %w", err)
	}
	defer rows.Close()
	var out []model.EmployeeStatistics
	for rows.Next() {
		var item model.EmployeeStatistics
		if err := rows.Scan(&item.EmployeeID, &item.Name, &item.Bookings, &item.Cancelled, &item.BookedMinutes, &item.PotentialRevenue); err != nil {
			return nil, fmt.Errorf("repository: scan employee statistics: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate employee statistics: %w", err)
	}
	return out, nil
}
