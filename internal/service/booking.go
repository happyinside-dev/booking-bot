package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/happyinside-dev/booking-bot/internal/apperr"
	"github.com/happyinside-dev/booking-bot/internal/model"
	"github.com/happyinside-dev/booking-bot/internal/repository"
)

type bookingRepository interface {
	Create(ctx context.Context, p repository.CreateBookingParams) (*model.Booking, error)
	GetByID(ctx context.Context, id int64) (*model.Booking, error)
	ListByUser(ctx context.Context, businessID, userID int64) ([]model.BookingView, error)
	ListByBusiness(ctx context.Context, businessID int64) ([]model.BookingView, error)
	Cancel(ctx context.Context, id, userID int64, now time.Time) error
}

type BookingService struct {
	repo       bookingRepository
	schedule   *ScheduleService
	services   scheduleServiceRepository
	employees  scheduleEmployeeRepository
	users      userRepository
	businesses scheduleBusinessRepository
	now        func() time.Time
}

func NewBookingService(repo bookingRepository, schedule *ScheduleService, services scheduleServiceRepository, employees scheduleEmployeeRepository, users userRepository, businesses scheduleBusinessRepository) *BookingService {
	return &BookingService{repo: repo, schedule: schedule, services: services, employees: employees, users: users, businesses: businesses, now: time.Now}
}

func (s *BookingService) ListServices(ctx context.Context, businessID int64) ([]model.Service, error) {
	items, err := s.services.ListByBusiness(ctx, businessID)
	if err != nil {
		return nil, err
	}
	out := items[:0]
	for _, v := range items {
		if v.IsActive {
			out = append(out, v)
		}
	}
	return out, nil
}
func (s *BookingService) ListEmployees(ctx context.Context, businessID int64) ([]model.Employee, error) {
	items, err := s.employees.ListByBusiness(ctx, businessID)
	if err != nil {
		return nil, err
	}
	out := items[:0]
	for _, v := range items {
		if v.IsActive {
			out = append(out, v)
		}
	}
	return out, nil
}
func (s *BookingService) AvailableSlots(ctx context.Context, employeeID, serviceID int64, date time.Time) ([]time.Time, error) {
	return s.schedule.GetAvailableSlots(ctx, AvailableSlotsRequest{EmployeeID: employeeID, ServiceID: serviceID, Date: date})
}

type CreateBookingInput struct {
	BusinessID, EmployeeID, ServiceID int64
	TelegramID                        int64
	StartAt                           time.Time
	Comment                           string
}

func (s *BookingService) Create(ctx context.Context, in CreateBookingInput) (*model.Booking, error) {
	user, err := s.users.GetByTelegramID(ctx, in.TelegramID)
	if err != nil {
		return nil, fmt.Errorf("service: booking user: %w", err)
	}
	business, err := s.businesses.GetByID(ctx, in.BusinessID)
	if err != nil {
		return nil, fmt.Errorf("service: booking business: %w", err)
	}
	loc, err := time.LoadLocation(business.Timezone)
	if err != nil {
		return nil, apperr.ErrInvalidTimezone
	}
	slots, err := s.schedule.GetAvailableSlots(ctx, AvailableSlotsRequest{EmployeeID: in.EmployeeID, ServiceID: in.ServiceID, Date: in.StartAt.In(loc)})
	if err != nil {
		return nil, err
	}
	valid := false
	for _, slot := range slots {
		if slot.Equal(in.StartAt.UTC()) {
			valid = true
			break
		}
	}
	if !valid {
		return nil, apperr.ErrInvalidTime
	}
	b, err := s.repo.Create(ctx, repository.CreateBookingParams{BusinessID: in.BusinessID, EmployeeID: in.EmployeeID, ServiceID: in.ServiceID, UserID: user.ID, StartAt: in.StartAt.UTC(), Comment: in.Comment})
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (s *BookingService) ListMine(ctx context.Context, telegramID, businessID int64) ([]model.BookingView, error) {
	u, err := s.users.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListByUser(ctx, businessID, u.ID)
}
func (s *BookingService) ListBusiness(ctx context.Context, actorUserID, businessID int64, auth RoleAuthorizer) ([]model.BookingView, error) {
	if err := authorizeAdmin(ctx, auth, businessID, actorUserID); err != nil {
		return nil, err
	}
	return s.repo.ListByBusiness(ctx, businessID)
}
func (s *BookingService) Cancel(ctx context.Context, telegramID, bookingID int64) error {
	u, err := s.users.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return err
	}
	b, err := s.repo.GetByID(ctx, bookingID)
	if err != nil {
		return err
	}
	if b.UserID != u.ID {
		return apperr.ErrForbidden
	}
	if b.Status == model.BookingCancelled {
		return apperr.ErrBookingAlreadyCancelled
	}
	if b.Status == model.BookingCompleted || b.Status == model.BookingNoShow || !b.StartAt.After(s.now()) {
		return apperr.ErrBookingAlreadyStarted
	}
	if err := s.repo.Cancel(ctx, b.ID, u.ID, s.now()); err != nil {
		if errors.Is(err, apperr.ErrBookingAlreadyStarted) {
			return err
		}
		return fmt.Errorf("service: cancel booking: %w", err)
	}
	return nil
}
