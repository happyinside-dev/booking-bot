package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/happyinside-dev/booking-bot/internal/apperr"
	"github.com/happyinside-dev/booking-bot/internal/model"
)

type BookingRepository struct{ pool *pgxpool.Pool }

func NewBookingRepository(pool *pgxpool.Pool) *BookingRepository {
	return &BookingRepository{pool: pool}
}

type CreateBookingParams struct {
	BusinessID, EmployeeID, ServiceID, UserID int64
	StartAt                                   time.Time
	Comment                                   string
}

func (r *BookingRepository) Create(ctx context.Context, p CreateBookingParams) (*model.Booking, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("repository: begin booking: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, p.EmployeeID); err != nil {
		return nil, fmt.Errorf("repository: lock employee: %w", err)
	}
	var duration int
	var price float64
	var serviceActive bool
	var serviceBusiness int64
	if err := tx.QueryRow(ctx, `SELECT business_id, duration_min, price, is_active FROM services WHERE id=$1 FOR SHARE`, p.ServiceID).Scan(&serviceBusiness, &duration, &price, &serviceActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("repository: load service for booking: %w", err)
	}
	var employeeBusiness int64
	var employeeActive bool
	if err := tx.QueryRow(ctx, `SELECT business_id, is_active FROM employees WHERE id=$1 FOR SHARE`, p.EmployeeID).Scan(&employeeBusiness, &employeeActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("repository: load employee for booking: %w", err)
	}
	if !serviceActive {
		return nil, apperr.ErrServiceInactive
	}
	if !employeeActive {
		return nil, apperr.ErrEmployeeInactive
	}
	if serviceBusiness != p.BusinessID || employeeBusiness != p.BusinessID {
		return nil, apperr.ErrForbidden
	}
	endAt := p.StartAt.Add(time.Duration(duration) * time.Minute)
	var blocked bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM blocked_slots WHERE employee_id=$1 AND start_at < $3 AND end_at > $2)`, p.EmployeeID, p.StartAt, endAt).Scan(&blocked); err != nil {
		return nil, fmt.Errorf("repository: check blocked slot: %w", err)
	}
	if blocked {
		return nil, apperr.ErrBookingConflict
	}
	b := &model.Booking{BusinessID: p.BusinessID, EmployeeID: p.EmployeeID, ServiceID: p.ServiceID, UserID: p.UserID, StartAt: p.StartAt.UTC(), EndAt: endAt.UTC(), Status: model.BookingConfirmed, Price: price, Comment: p.Comment}
	const q = `INSERT INTO bookings (business_id,employee_id,service_id,user_id,start_at,end_at,status,price,comment)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id,created_at,updated_at`
	if err := tx.QueryRow(ctx, q, b.BusinessID, b.EmployeeID, b.ServiceID, b.UserID, b.StartAt, b.EndAt, b.Status, b.Price, b.Comment).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, mapBookingWriteError("insert", err)
	}
	const notificationQ = `INSERT INTO notifications (booking_id,user_id,type,scheduled_at)
		SELECT $1,$2,v.type,v.scheduled_at FROM (VALUES
			('REMINDER_24H'::varchar,$3::timestamptz - interval '24 hours'),
			('REMINDER_1H'::varchar,$3::timestamptz - interval '1 hour')
		) AS v(type,scheduled_at) WHERE v.scheduled_at > now()`
	if _, err := tx.Exec(ctx, notificationQ, b.ID, b.UserID, b.StartAt); err != nil {
		return nil, fmt.Errorf("repository: create booking notifications: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, mapBookingWriteError("commit", err)
	}
	return b, nil
}

func mapBookingWriteError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23P01" {
		return apperr.ErrBookingConflict
	}
	return fmt.Errorf("repository: %s booking: %w", operation, err)
}

func (r *BookingRepository) GetByID(ctx context.Context, id int64) (*model.Booking, error) {
	const q = `SELECT id,business_id,employee_id,service_id,user_id,start_at,end_at,status,price,comment,created_at,updated_at FROM bookings WHERE id=$1`
	var b model.Booking
	if err := r.pool.QueryRow(ctx, q, id).Scan(&b.ID, &b.BusinessID, &b.EmployeeID, &b.ServiceID, &b.UserID, &b.StartAt, &b.EndAt, &b.Status, &b.Price, &b.Comment, &b.CreatedAt, &b.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("repository: get booking: %w", err)
	}
	return &b, nil
}

func (r *BookingRepository) ListByUser(ctx context.Context, businessID, userID int64) ([]model.BookingView, error) {
	return r.list(ctx, businessID, `b.user_id=$2`, userID)
}
func (r *BookingRepository) ListByBusiness(ctx context.Context, businessID int64) ([]model.BookingView, error) {
	return r.list(ctx, businessID, `TRUE`, 0)
}
func (r *BookingRepository) list(ctx context.Context, businessID int64, condition string, arg int64) ([]model.BookingView, error) {
	q := `SELECT b.id,b.business_id,b.employee_id,b.service_id,b.user_id,b.start_at,b.end_at,b.status,b.price,b.comment,b.created_at,b.updated_at,
		s.name,e.name,biz.name,biz.timezone FROM bookings b JOIN services s ON s.id=b.service_id JOIN employees e ON e.id=b.employee_id JOIN businesses biz ON biz.id=b.business_id
		WHERE b.business_id=$1 AND ` + condition + ` ORDER BY b.start_at DESC LIMIT 100`
	var rows pgx.Rows
	var err error
	if condition == `TRUE` {
		rows, err = r.pool.Query(ctx, q, businessID)
	} else {
		rows, err = r.pool.Query(ctx, q, businessID, arg)
	}
	if err != nil {
		return nil, fmt.Errorf("repository: list bookings: %w", err)
	}
	defer rows.Close()
	var out []model.BookingView
	for rows.Next() {
		var v model.BookingView
		if err := rows.Scan(&v.ID, &v.BusinessID, &v.EmployeeID, &v.ServiceID, &v.UserID, &v.StartAt, &v.EndAt, &v.Status, &v.Price, &v.Comment, &v.CreatedAt, &v.UpdatedAt, &v.ServiceName, &v.EmployeeName, &v.BusinessName, &v.BusinessTimezone); err != nil {
			return nil, fmt.Errorf("repository: scan booking: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *BookingRepository) Cancel(ctx context.Context, id, userID int64, now time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repository: begin cancel booking: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE bookings SET status='CANCELLED',updated_at=$3 WHERE id=$1 AND user_id=$2 AND start_at>$3 AND status IN ('PENDING','CONFIRMED')`, id, userID, now)
	if err != nil {
		return fmt.Errorf("repository: cancel booking: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrBookingAlreadyStarted
	}
	if _, err := tx.Exec(ctx, `UPDATE notifications SET status='CANCELLED',locked_until=NULL,updated_at=$2 WHERE booking_id=$1 AND status IN ('PENDING','PROCESSING')`, id, now); err != nil {
		return fmt.Errorf("repository: cancel notifications: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("repository: commit booking cancellation: %w", err)
	}
	return nil
}
