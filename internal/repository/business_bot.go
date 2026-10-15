package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/happyinside-dev/booking-bot/internal/model"
)

type BusinessBotRepository struct{ pool *pgxpool.Pool }

func NewBusinessBotRepository(pool *pgxpool.Pool) *BusinessBotRepository { return &BusinessBotRepository{pool: pool} }

func (r *BusinessBotRepository) ListActive(ctx context.Context) ([]model.BusinessBot, error) {
	const q = `SELECT id, business_id, username, token_key, is_active, created_at, updated_at
		FROM business_bots WHERE is_active = TRUE ORDER BY id`
	rows, err := r.pool.Query(ctx, q)
	if err != nil { return nil, fmt.Errorf("repository: list business bots: %w", err) }
	defer rows.Close()
	var result []model.BusinessBot
	for rows.Next() {
		var item model.BusinessBot
		if err := rows.Scan(&item.ID, &item.BusinessID, &item.Username, &item.TokenKey, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan business bot: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil { return nil, fmt.Errorf("repository: iterate business bots: %w", err) }
	return result, nil
}
