package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/happyinside-dev/booking-bot/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct{ pool *pgxpool.Pool }

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

func (r *NotificationRepository) ClaimDue(ctx context.Context, now time.Time, limit int, lease time.Duration) ([]model.Notification, error) {
	const q = `WITH candidates AS (
		SELECT n.id FROM notifications n JOIN bookings b ON b.id=n.booking_id
		WHERE b.status IN ('PENDING','CONFIRMED') AND n.scheduled_at <= $1
		  AND (n.status='PENDING' OR (n.status='PROCESSING' AND n.locked_until < $1))
		ORDER BY n.scheduled_at,n.id FOR UPDATE OF n SKIP LOCKED LIMIT $2
	), claimed AS (
		UPDATE notifications n SET status='PROCESSING',attempts=n.attempts+1,locked_until=$1+$3::interval,updated_at=$1
		FROM candidates c WHERE n.id=c.id RETURNING n.*
	)
	SELECT c.id,c.booking_id,c.user_id,c.type,c.scheduled_at,c.sent_at,c.status,c.attempts,c.locked_until,c.last_error,c.created_at,c.updated_at,
	       b.business_id,u.telegram_id,b.start_at,b.price,s.name,e.name,biz.timezone
	FROM claimed c JOIN bookings b ON b.id=c.booking_id JOIN users u ON u.id=c.user_id
	JOIN services s ON s.id=b.service_id JOIN employees e ON e.id=b.employee_id JOIN businesses biz ON biz.id=b.business_id
	ORDER BY c.scheduled_at,c.id`
	rows, err := r.pool.Query(ctx, q, now, limit, fmt.Sprintf("%f seconds", lease.Seconds()))
	if err != nil {
		return nil, fmt.Errorf("repository: claim notifications: %w", err)
	}
	defer rows.Close()
	var out []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.BookingID, &n.UserID, &n.Type, &n.ScheduledAt, &n.SentAt, &n.Status, &n.Attempts, &n.LockedUntil, &n.LastError, &n.CreatedAt, &n.UpdatedAt, &n.BusinessID, &n.TelegramID, &n.BookingStart, &n.Price, &n.ServiceName, &n.EmployeeName, &n.BusinessTimezone); err != nil {
			return nil, fmt.Errorf("repository: scan notification: %w", err)
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate notifications: %w", err)
	}
	return out, nil
}

func (r *NotificationRepository) IsDeliverable(ctx context.Context, id int64) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM notifications n JOIN bookings b ON b.id=n.booking_id WHERE n.id=$1 AND n.status='PROCESSING' AND b.status IN ('PENDING','CONFIRMED'))`, id).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("repository: check notification: %w", err)
	}
	return ok, nil
}
func (r *NotificationRepository) MarkSent(ctx context.Context, id int64, sentAt time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE notifications SET status='SENT',sent_at=$2,locked_until=NULL,last_error=NULL,updated_at=$2 WHERE id=$1 AND status='PROCESSING'`, id, sentAt)
	if err != nil {
		return fmt.Errorf("repository: mark notification sent: %w", err)
	}
	return nil
}
func (r *NotificationRepository) MarkFailed(ctx context.Context, id int64, now time.Time, maxAttempts int, retryDelay time.Duration, message string) error {
	const q = `UPDATE notifications SET status=CASE WHEN attempts >= $3 THEN 'FAILED' ELSE 'PENDING' END,
		scheduled_at=CASE WHEN attempts >= $3 THEN scheduled_at ELSE $2+$4::interval END,locked_until=NULL,last_error=$5,updated_at=$2 WHERE id=$1 AND status='PROCESSING'`
	_, err := r.pool.Exec(ctx, q, id, now, maxAttempts, fmt.Sprintf("%f seconds", retryDelay.Seconds()), message)
	if err != nil {
		return fmt.Errorf("repository: mark notification failed: %w", err)
	}
	return nil
}
