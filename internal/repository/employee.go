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

type EmployeeRepository struct {
	pool *pgxpool.Pool
}

func NewEmployeeRepository(pool *pgxpool.Pool) *EmployeeRepository {
	return &EmployeeRepository{pool: pool}
}

func (r *EmployeeRepository) Create(ctx context.Context, e *model.Employee) (*model.Employee, error) {
	const q = `
	INSERT INTO employees
	(business_id,user_id,name,description,is_active,created_at,updated_at)
	VALUES($1,$2,$3,$4,TRUE,now(),now())
	RETURNING id,is_active,created_at,updated_at;
	`

	err := r.pool.QueryRow(
		ctx,
		q,
		e.BusinessID,
		e.UserID,
		e.Name,
		e.Description,
	).Scan(
		&e.ID,
		&e.IsActive,
		&e.CreatedAt,
		&e.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("repository: create employee: %w", err)
	}

	return e, nil
}

func (r *EmployeeRepository) GetByID(ctx context.Context, id int64) (*model.Employee, error) {
	const q = `
	SELECT id,business_id,user_id,name,description,is_active,created_at,updated_at
	FROM employees
	WHERE id=$1;
	`

	var e model.Employee

	err := r.pool.QueryRow(ctx, q, id).Scan(
		&e.ID,
		&e.BusinessID,
		&e.UserID,
		&e.Name,
		&e.Description,
		&e.IsActive,
		&e.CreatedAt,
		&e.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("repository: get employee: %w", err)
	}

	return &e, nil
}

func (r *EmployeeRepository) ListByBusiness(ctx context.Context, businessID int64) ([]model.Employee, error) {
	const q = `
	SELECT id,business_id,user_id,name,description,is_active,created_at,updated_at
	FROM employees
	WHERE business_id=$1
	ORDER BY name;
	`

	rows, err := r.pool.Query(ctx, q, businessID)
	if err != nil {
		return nil, fmt.Errorf("repository: list employees: %w", err)
	}
	defer rows.Close()

	var employees []model.Employee

	for rows.Next() {
		var e model.Employee

		err := rows.Scan(
			&e.ID,
			&e.BusinessID,
			&e.UserID,
			&e.Name,
			&e.Description,
			&e.IsActive,
			&e.CreatedAt,
			&e.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("repository: scan employee: %w", err)
		}

		employees = append(employees, e)
	}

	return employees, rows.Err()
}

func (r *EmployeeRepository) Update(ctx context.Context, e *model.Employee) error {
	const q = `
	UPDATE employees
	SET
		user_id=$2,
		name=$3,
		description=$4,
		is_active=$5,
		updated_at=now()
	WHERE id=$1;
	`

	tag, err := r.pool.Exec(
		ctx,
		q,
		e.ID,
		e.UserID,
		e.Name,
		e.Description,
		e.IsActive,
	)

	if err != nil {
		return fmt.Errorf("repository: update employee: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}

	return nil
}

