package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/happyinside-dev/booking-bot/internal/apperr"
	"github.com/happyinside-dev/booking-bot/internal/model"
)

type BusinessRepository struct {
	pool *pgxpool.Pool
}

func NewBusinessRepository(pool *pgxpool.Pool) *BusinessRepository {
	return &BusinessRepository{pool: pool}
}

// CreateWithOwner atomically creates a business and its OWNER membership.
// A business without an owner is not a valid domain state.
func (r *BusinessRepository) CreateWithOwner(ctx context.Context, b *model.Business, ownerUserID int64) (*model.Business, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin create business: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO businesses (
			name, description, address, timezone,
			slot_interval_min, min_booking_notice_min, max_booking_days,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now())
		RETURNING id, created_at, updated_at
	`

	err = tx.QueryRow(
		ctx,
		q,
		b.Name,
		b.Description,
		b.Address,
		b.Timezone,
		b.SlotIntervalMin,
		b.MinBookingNoticeMin,
		b.MaxBookingDays,
	).Scan(
		&b.ID,
		&b.CreatedAt,
		&b.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("repository: create business: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO business_members (business_id,user_id,role) VALUES ($1,$2,$3)`, b.ID, ownerUserID, model.RoleOwner); err != nil {
		return nil, fmt.Errorf("repository: add business owner: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit create business: %w", err)
	}

	return b, nil
}

func (r *BusinessRepository) GetByID(ctx context.Context, id int64) (*model.Business, error) {
	const q = `
		SELECT id, name, description, address, timezone,
		       slot_interval_min, min_booking_notice_min, max_booking_days,
		       created_at, updated_at
		FROM businesses
		WHERE id = $1
	`

	var b model.Business

	err := r.pool.QueryRow(ctx, q, id).Scan(
		&b.ID,
		&b.Name,
		&b.Description,
		&b.Address,
		&b.Timezone,
		&b.SlotIntervalMin,
		&b.MinBookingNoticeMin,
		&b.MaxBookingDays,
		&b.CreatedAt,
		&b.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("repository: get business: %w", err)
	}

	return &b, nil
}

func (r *BusinessRepository) SingleBusinessID(ctx context.Context) (int64, error) {
	rows, err := r.pool.Query(ctx, `SELECT id FROM businesses ORDER BY id LIMIT 2`)
	if err != nil {
		return 0, fmt.Errorf("repository: list businesses: %w", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("repository: scan business: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("repository: iterate businesses: %w", err)
	}
	if len(ids) == 0 {
		return 0, fmt.Errorf("no businesses configured")
	}
	if len(ids) > 1 {
		return 0, fmt.Errorf("multiple businesses configured: BOT_BUSINESS_ID is required in single-bot mode")
	}
	return ids[0], nil
}
