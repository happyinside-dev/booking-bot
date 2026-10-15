package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/happyinside-dev/booking-bot/internal/apperr"
	"github.com/happyinside-dev/booking-bot/internal/model"
)

type businessRepository interface {
	Create(ctx context.Context, b *model.Business) (*model.Business, error)
	GetByID(ctx context.Context, id int64) (*model.Business, error)
}

type businessMemberRepository interface {
	Add(ctx context.Context, m *model.BusinessMember) (*model.BusinessMember, error)
	GetRole(ctx context.Context, businessID, userID int64) (model.Role, error)
	ListBusinesses(ctx context.Context, userID int64) ([]int64, error)
}

type BusinessService struct {
	businessRepo businessRepository
	memberRepo   businessMemberRepository
	log          *slog.Logger
	defaults     BusinessDefaults
}

type BusinessDefaults struct {
	SlotIntervalMin     int
	MinBookingNoticeMin int
	MaxBookingDays      int
}

func NewBusinessService(
	businessRepo businessRepository,
	memberRepo businessMemberRepository,
	log *slog.Logger,
	defaults BusinessDefaults,
) *BusinessService {
	return &BusinessService{
		businessRepo: businessRepo,
		memberRepo:   memberRepo,
		log:          log,
		defaults:     defaults,
	}
}

type CreateBusinessInput struct {
	Name        string
	Description string
	Address     string
	Timezone    string
	OwnerUserID int64
}

// CreateBusiness создаёт бизнес и сразу добавляет владельца.
func (s *BusinessService) CreateBusiness(
	ctx context.Context,
	in CreateBusinessInput,
) (*model.Business, error) {

	business := &model.Business{
		Name:                in.Name,
		Description:         in.Description,
		Address:             in.Address,
		Timezone:            in.Timezone,
		SlotIntervalMin:     s.defaults.SlotIntervalMin,
		MinBookingNoticeMin: s.defaults.MinBookingNoticeMin,
		MaxBookingDays:      s.defaults.MaxBookingDays,
	}

	createdBusiness, err := s.businessRepo.Create(ctx, business)
	if err != nil {
		return nil, fmt.Errorf("service: create business: %w", err)
	}

	member := &model.BusinessMember{
		BusinessID: createdBusiness.ID,
		UserID:     in.OwnerUserID,
		Role:       model.RoleOwner,
	}

	_, err = s.memberRepo.Add(ctx, member)
	if err != nil {
		return nil, fmt.Errorf("service: add owner: %w", err)
	}

	s.log.Info(
		"business created",
		slog.Int64("business_id", createdBusiness.ID),
		slog.Int64("owner_id", in.OwnerUserID),
	)

	return createdBusiness, nil
}

func (s *BusinessService) AddMember(
	ctx context.Context,
	businessID, userID int64,
	role model.Role,
) error {

	member := &model.BusinessMember{
		BusinessID: businessID,
		UserID:     userID,
		Role:       role,
	}

	_, err := s.memberRepo.Add(ctx, member)
	if err != nil {
		return fmt.Errorf("service: add member: %w", err)
	}

	s.log.Info(
		"member added",
		slog.Int64("business_id", businessID),
		slog.Int64("user_id", userID),
		slog.String("role", string(role)),
	)

	return nil
}

// HasRole проверяет, есть ли у пользователя одна из разрешённых ролей.
func (s *BusinessService) HasRole(
	ctx context.Context,
	businessID, userID int64,
	allowed ...model.Role,
) error {

	role, err := s.memberRepo.GetRole(ctx, businessID, userID)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return apperr.ErrForbidden
		}
		return fmt.Errorf("service: get role: %w", err)
	}

	for _, r := range allowed {
		if role == r {
			return nil
		}
	}

	return apperr.ErrForbidden
}

// FirstBusiness возвращает первый бизнес пользователя.
// Пока этого достаточно для MVP.
func (s *BusinessService) FirstBusiness(
	ctx context.Context,
	userID int64,
) (int64, error) {

	businesses, err := s.memberRepo.ListBusinesses(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("service: list businesses: %w", err)
	}

	if len(businesses) == 0 {
		return 0, apperr.ErrNotFound
	}

	return businesses[0], nil
}

func (s *BusinessService) GetByID(ctx context.Context, id int64) (*model.Business, error) {
	business, err := s.businessRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service: get business: %w", err)
	}
	return business, nil
}
