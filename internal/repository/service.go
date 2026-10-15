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

type ServiceRepository struct {
	pool *pgxpool.Pool
}

func NewServiceRepository(pool *pgxpool.Pool) *ServiceRepository {
	return &ServiceRepository{pool: pool}
}

func (r *ServiceRepository) Create(ctx context.Context, s *model.Service) (*model.Service, error) {
	const q = `
	INSERT INTO services
	(business_id,name,description,duration_min,price,is_active,created_at,updated_at)
	VALUES($1,$2,$3,$4,$5,TRUE,now(),now())
	RETURNING id,is_active,created_at,updated_at;
	`

	err := r.pool.QueryRow(
		ctx,
		q,
		s.BusinessID,
		s.Name,
		s.Description,
		s.DurationMin,
		s.Price,
	).Scan(
		&s.ID,
		&s.IsActive,
		&s.CreatedAt,
		&s.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("repository: create service: %w", err)
	}

	return s, nil
}

func (r *ServiceRepository) GetByID(ctx context.Context, id int64) (*model.Service, error) {
	const q = `
	SELECT id,business_id,name,description,duration_min,price,is_active,created_at,updated_at
	FROM services
	WHERE id=$1;
	`

	var s model.Service

	err := r.pool.QueryRow(ctx, q, id).Scan(
		&s.ID,
		&s.BusinessID,
		&s.Name,
		&s.Description,
		&s.DurationMin,
		&s.Price,
		&s.IsActive,
		&s.CreatedAt,
		&s.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("repository: get service: %w", err)
	}

	return &s, nil
}

func (r *ServiceRepository) ListByBusiness(ctx context.Context, businessID int64) ([]model.Service, error) {
	const q = `
	SELECT id,business_id,name,description,duration_min,price,is_active,created_at,updated_at
	FROM services
	WHERE business_id=$1
	ORDER BY name;
	`

	rows, err := r.pool.Query(ctx, q, businessID)
	if err != nil {
		return nil, fmt.Errorf("repository: list services: %w", err)
	}
	defer rows.Close()

	var services []model.Service

	for rows.Next() {
		var s model.Service

		err := rows.Scan(
			&s.ID,
			&s.BusinessID,
			&s.Name,
			&s.Description,
			&s.DurationMin,
			&s.Price,
			&s.IsActive,
			&s.CreatedAt,
			&s.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("repository: scan service: %w", err)
		}

		services = append(services, s)
	}

	return services, rows.Err()
}

func (r *ServiceRepository) Update(ctx context.Context, s *model.Service) error {
	const q = `
	UPDATE services
	SET
		name=$2,
		description=$3,
		duration_min=$4,
		price=$5,
		is_active=$6,
		updated_at=now()
	WHERE id=$1;
	`

	tag, err := r.pool.Exec(
		ctx,
		q,
		s.ID,
		s.Name,
		s.Description,
		s.DurationMin,
		s.Price,
		s.IsActive,
	)

	if err != nil {
		return fmt.Errorf("repository: update service: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}

	return nil
}

// Delete is intentionally absent: services are retained for booking history
// and deactivated through IsActive instead.
