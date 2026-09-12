package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourname/booking-bot/internal/apperr"
	"github.com/yourname/booking-bot/internal/model"
)

type BusinessRepository struct {
	pool *pgxpool.Pool
}

func NewBusinessRepository(pool *pgxpool.Pool) *BusinessRepository {
	return &BusinessRepository{pool: pool}
}

func (r *BusinessRepository) Create(ctx context.Context, b *model.Business) (*model.Business, error) {
	const q = `
		INSERT INTO businesses (name, description, address, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, now(), now())
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(
		ctx,
		q,
		b.Name,
		b.Description,
		b.Address,
		b.Timezone,
	).Scan(
		&b.ID,
		&b.CreatedAt,
		&b.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("repository: create business: %w", err)
	}

	return b, nil
}

func (r *BusinessRepository) GetByID(ctx context.Context, id int64) (*model.Business, error) {
	const q = `
		SELECT id, name, description, address, timezone, created_at, updated_at
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
