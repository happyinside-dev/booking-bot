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

type BusinessMemberRepository struct {
	pool *pgxpool.Pool
}

func NewBusinessMemberRepository(pool *pgxpool.Pool) *BusinessMemberRepository {
	return &BusinessMemberRepository{pool: pool}
}

func (r *BusinessMemberRepository) Add(ctx context.Context, m *model.BusinessMember) (*model.BusinessMember, error) {
	const q = `
		INSERT INTO business_members (business_id, user_id, role, created_at)
		VALUES ($1, $2, $3, now())
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(
		ctx,
		q,
		m.BusinessID,
		m.UserID,
		m.Role,
	).Scan(
		&m.ID,
		&m.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("repository: add business member: %w", err)
	}

	return m, nil
}

func (r *BusinessMemberRepository) GetRole(ctx context.Context, businessID, userID int64) (model.Role, error) {
	const q = `
		SELECT role
		FROM business_members
		WHERE business_id = $1
		  AND user_id = $2
	`

	var role model.Role

	err := r.pool.QueryRow(ctx, q, businessID, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", apperr.ErrNotFound
		}
		return "", fmt.Errorf("repository: get member role: %w", err)
	}

	return role, nil
}

func (r *BusinessMemberRepository) ListBusinesses(ctx context.Context, userID int64) ([]int64, error) {
	const q = `
		SELECT business_id
		FROM business_members
		WHERE user_id = $1
		ORDER BY business_id
	`

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("repository: list businesses: %w", err)
	}
	defer rows.Close()

	var ids []int64

	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("repository: scan business_id: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate businesses: %w", err)
	}

	return ids, nil
}
