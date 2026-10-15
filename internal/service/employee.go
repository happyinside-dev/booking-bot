package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/happyinside-dev/booking-bot/internal/apperr"
	"github.com/happyinside-dev/booking-bot/internal/model"
)

type employeeRepository interface {
	Create(ctx context.Context, employee *model.Employee) (*model.Employee, error)
	GetByID(ctx context.Context, id int64) (*model.Employee, error)
	ListByBusiness(ctx context.Context, businessID int64) ([]model.Employee, error)
	Update(ctx context.Context, employee *model.Employee) error
}

type EmployeeService struct {
	repo employeeRepository
	auth RoleAuthorizer
}

func NewEmployeeService(repo employeeRepository, auth RoleAuthorizer) *EmployeeService {
	return &EmployeeService{repo: repo, auth: auth}
}

type CreateEmployeeInput struct {
	ActorUserID, BusinessID int64
	UserID                  *int64
	Name, Description       string
}

func (s *EmployeeService) Create(ctx context.Context, in CreateEmployeeInput) (*model.Employee, error) {
	if err := authorizeAdmin(ctx, s.auth, in.BusinessID, in.ActorUserID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.Create(ctx, &model.Employee{BusinessID: in.BusinessID, UserID: in.UserID, Name: strings.TrimSpace(in.Name), Description: strings.TrimSpace(in.Description)})
}

type UpdateEmployeeInput struct {
	ActorUserID, BusinessID, EmployeeID int64
	UserID                              *int64
	Name, Description                   string
}

func (s *EmployeeService) Update(ctx context.Context, in UpdateEmployeeInput) (*model.Employee, error) {
	if err := authorizeAdmin(ctx, s.auth, in.BusinessID, in.ActorUserID); err != nil {
		return nil, err
	}
	employee, err := s.owned(ctx, in.BusinessID, in.EmployeeID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, apperr.ErrInvalidInput
	}
	employee.UserID, employee.Name, employee.Description = in.UserID, strings.TrimSpace(in.Name), strings.TrimSpace(in.Description)
	if err := s.repo.Update(ctx, employee); err != nil {
		return nil, fmt.Errorf("service: update employee: %w", err)
	}
	return employee, nil
}

func (s *EmployeeService) SetActive(ctx context.Context, actorUserID, businessID, employeeID int64, active bool) error {
	if err := authorizeAdmin(ctx, s.auth, businessID, actorUserID); err != nil {
		return err
	}
	employee, err := s.owned(ctx, businessID, employeeID)
	if err != nil {
		return err
	}
	employee.IsActive = active
	if err := s.repo.Update(ctx, employee); err != nil {
		return fmt.Errorf("service: set employee active: %w", err)
	}
	return nil
}

func (s *EmployeeService) List(ctx context.Context, actorUserID, businessID int64) ([]model.Employee, error) {
	if err := authorizeAdmin(ctx, s.auth, businessID, actorUserID); err != nil {
		return nil, err
	}
	employees, err := s.repo.ListByBusiness(ctx, businessID)
	if err != nil {
		return nil, fmt.Errorf("service: list employees: %w", err)
	}
	return employees, nil
}

func (s *EmployeeService) owned(ctx context.Context, businessID, id int64) (*model.Employee, error) {
	employee, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service: get employee: %w", err)
	}
	if employee.BusinessID != businessID {
		return nil, apperr.ErrForbidden
	}
	return employee, nil
}
