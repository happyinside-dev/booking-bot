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

// UserRepository provides access to the users table. It knows nothing about
// Telegram or business rules -- it only maps rows to model.User.
type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*model.User, error) {
	const q = `
		SELECT id, telegram_id, username, first_name, last_name, phone, created_at, updated_at
		FROM users
		WHERE telegram_id = $1
	`

	var u model.User
	err := r.pool.QueryRow(ctx, q, telegramID).Scan(
		&u.ID, &u.TelegramID, &u.Username, &u.FirstName, &u.LastName, &u.Phone,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("repository: get user by telegram_id: %w", err)
	}

	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, u *model.User) (*model.User, error) {
	const q = `
		INSERT INTO users (telegram_id, username, first_name, last_name, phone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, now(), now())
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, q, u.TelegramID, u.Username, u.FirstName, u.LastName, u.Phone).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("repository: create user: %w", err)
	}

	return u, nil
}

// Update overwrites the mutable profile fields of an existing user
// (identified by ID) and refreshes updated_at.
func (r *UserRepository) Update(ctx context.Context, u *model.User) error {
	const q = `
		UPDATE users
		SET username = $1, first_name = $2, last_name = $3, phone = $4, updated_at = now()
		WHERE id = $5
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, q, u.Username, u.FirstName, u.LastName, u.Phone, u.ID).
		Scan(&u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.ErrNotFound
		}
		return fmt.Errorf("repository: update user: %w", err)
	}

	return nil
}
