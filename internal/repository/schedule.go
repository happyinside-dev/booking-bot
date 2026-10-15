package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/happyinside-dev/booking-bot/internal/apperr"
	"github.com/happyinside-dev/booking-bot/internal/model"
)

type ScheduleRepository struct{ pool *pgxpool.Pool }

func NewScheduleRepository(pool *pgxpool.Pool) *ScheduleRepository {
	return &ScheduleRepository{pool: pool}
}

func (r *ScheduleRepository) UpsertWorkingHours(ctx context.Context, h *model.WorkingHours) (*model.WorkingHours, error) {
	const q = `
		INSERT INTO working_hours (employee_id, weekday, start_time, end_time, is_day_off)
		VALUES ($1, $2, ($3 * interval '1 minute')::time, ($4 * interval '1 minute')::time, $5)
		ON CONFLICT (employee_id, weekday) DO UPDATE SET
			start_time = EXCLUDED.start_time, end_time = EXCLUDED.end_time, is_day_off = EXCLUDED.is_day_off
		RETURNING id`
	if err := r.pool.QueryRow(ctx, q, h.EmployeeID, h.Weekday, h.StartMinute, h.EndMinute, h.IsDayOff).Scan(&h.ID); err != nil {
		return nil, fmt.Errorf("repository: upsert working hours: %w", err)
	}
	return h, nil
}

func (r *ScheduleRepository) GetWorkingHours(ctx context.Context, employeeID int64, weekday int) (*model.WorkingHours, error) {
	const q = `
		SELECT id, employee_id, weekday,
		       (EXTRACT(EPOCH FROM start_time) / 60)::int,
		       (EXTRACT(EPOCH FROM end_time) / 60)::int, is_day_off
		FROM working_hours WHERE employee_id = $1 AND weekday = $2`
	var h model.WorkingHours
	if err := r.pool.QueryRow(ctx, q, employeeID, weekday).Scan(
		&h.ID, &h.EmployeeID, &h.Weekday, &h.StartMinute, &h.EndMinute, &h.IsDayOff,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("repository: get working hours: %w", err)
	}
	return &h, nil
}

func (r *ScheduleRepository) CreateBlockedSlot(ctx context.Context, slot *model.BlockedSlot) (*model.BlockedSlot, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin blocked slot: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, slot.EmployeeID); err != nil {
		return nil, fmt.Errorf("repository: lock employee: %w", err)
	}
	var conflict bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM bookings WHERE employee_id=$1 AND status IN ('PENDING','CONFIRMED') AND start_at < $3 AND end_at > $2)`, slot.EmployeeID, slot.StartAt, slot.EndAt).Scan(&conflict); err != nil {
		return nil, fmt.Errorf("repository: check bookings for blocked slot: %w", err)
	}
	if conflict {
		return nil, apperr.ErrBookingConflict
	}
	const q = `
		INSERT INTO blocked_slots (employee_id, start_at, end_at, reason)
		VALUES ($1, $2, $3, NULLIF($4, '')) RETURNING id, created_at`
	if err := tx.QueryRow(ctx, q, slot.EmployeeID, slot.StartAt, slot.EndAt, slot.Reason).Scan(&slot.ID, &slot.CreatedAt); err != nil {
		return nil, fmt.Errorf("repository: create blocked slot: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit blocked slot: %w", err)
	}
	return slot, nil
}

// ListBusyIntervals returns all stage-5 busy intervals. Active bookings are
// added to this query in stage 6, after the bookings table is introduced.
func (r *ScheduleRepository) ListBusyIntervals(ctx context.Context, employeeID int64, from, to time.Time) ([]model.TimeInterval, error) {
	const q = `
		SELECT start_at, end_at FROM (
			SELECT start_at, end_at FROM blocked_slots WHERE employee_id=$1 AND start_at<$3 AND end_at>$2
			UNION ALL
			SELECT start_at, end_at FROM bookings WHERE employee_id=$1 AND status IN ('PENDING','CONFIRMED') AND start_at<$3 AND end_at>$2
		) busy
		ORDER BY start_at, end_at`
	rows, err := r.pool.Query(ctx, q, employeeID, from, to)
	if err != nil {
		return nil, fmt.Errorf("repository: list busy intervals: %w", err)
	}
	defer rows.Close()
	var result []model.TimeInterval
	for rows.Next() {
		var interval model.TimeInterval
		if err := rows.Scan(&interval.Start, &interval.End); err != nil {
			return nil, fmt.Errorf("repository: scan busy interval: %w", err)
		}
		result = append(result, interval)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate busy intervals: %w", err)
	}
	return result, nil
}
