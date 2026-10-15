package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/happyinside-dev/booking-bot/internal/apperr"
	"github.com/happyinside-dev/booking-bot/internal/model"
)

type catalogRepository interface {
	Create(ctx context.Context, item *model.Service) (*model.Service, error)
	GetByID(ctx context.Context, id int64) (*model.Service, error)
	ListByBusiness(ctx context.Context, businessID int64) ([]model.Service, error)
	Update(ctx context.Context, item *model.Service) error
}

type CatalogService struct {
	repo catalogRepository
	auth RoleAuthorizer
}

func NewCatalogService(repo catalogRepository, auth RoleAuthorizer) *CatalogService {
	return &CatalogService{repo: repo, auth: auth}
}

type CreateServiceInput struct {
	ActorUserID int64
	BusinessID  int64
	Name        string
	Description string
	DurationMin int
	Price       float64
}

func (s *CatalogService) Create(ctx context.Context, in CreateServiceInput) (*model.Service, error) {
	if err := authorizeAdmin(ctx, s.auth, in.BusinessID, in.ActorUserID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) == "" || in.DurationMin <= 0 || in.Price < 0 {
		return nil, apperr.ErrInvalidInput
	}
	return s.repo.Create(ctx, &model.Service{
		BusinessID: in.BusinessID, Name: strings.TrimSpace(in.Name), Description: strings.TrimSpace(in.Description),
		DurationMin: in.DurationMin, Price: in.Price,
	})
}

type UpdateServiceInput struct {
	ActorUserID int64
	BusinessID  int64
	ServiceID   int64
	Name        string
	Description string
	DurationMin int
	Price       float64
}

func (s *CatalogService) Update(ctx context.Context, in UpdateServiceInput) (*model.Service, error) {
	if err := authorizeAdmin(ctx, s.auth, in.BusinessID, in.ActorUserID); err != nil {
		return nil, err
	}
	item, err := s.owned(ctx, in.BusinessID, in.ServiceID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) == "" || in.DurationMin <= 0 || in.Price < 0 {
		return nil, apperr.ErrInvalidInput
	}
	item.Name, item.Description, item.DurationMin, item.Price = strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), in.DurationMin, in.Price
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, fmt.Errorf("service: update catalog item: %w", err)
	}
	return item, nil
}

func (s *CatalogService) SetActive(ctx context.Context, actorUserID, businessID, serviceID int64, active bool) error {
	if err := authorizeAdmin(ctx, s.auth, businessID, actorUserID); err != nil {
		return err
	}
	item, err := s.owned(ctx, businessID, serviceID)
	if err != nil {
		return err
	}
	item.IsActive = active
	if err := s.repo.Update(ctx, item); err != nil {
		return fmt.Errorf("service: set service active: %w", err)
	}
	return nil
}

func (s *CatalogService) List(ctx context.Context, actorUserID, businessID int64) ([]model.Service, error) {
	if err := authorizeAdmin(ctx, s.auth, businessID, actorUserID); err != nil {
		return nil, err
	}
	items, err := s.repo.ListByBusiness(ctx, businessID)
	if err != nil {
		return nil, fmt.Errorf("service: list services: %w", err)
	}
	return items, nil
}

func (s *CatalogService) owned(ctx context.Context, businessID, id int64) (*model.Service, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service: get service: %w", err)
	}
	if item.BusinessID != businessID {
		return nil, apperr.ErrForbidden
	}
	return item, nil
}
