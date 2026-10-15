package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/happyinside-dev/booking-bot/internal/apperr"
	"github.com/happyinside-dev/booking-bot/internal/model"
	"github.com/happyinside-dev/booking-bot/internal/repository"
)

type bookingRepoFake struct {
	created   *repository.CreateBookingParams
	createErr error
	booking   *model.Booking
	cancelled bool
}

func (f *bookingRepoFake) Create(_ context.Context, p repository.CreateBookingParams) (*model.Booking, error) {
	f.created = &p
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &model.Booking{ID: 1, StartAt: p.StartAt}, nil
}
func (f *bookingRepoFake) GetByID(context.Context, int64) (*model.Booking, error) {
	if f.booking == nil {
		return nil, apperr.ErrNotFound
	}
	return f.booking, nil
}
func (f *bookingRepoFake) ListByUser(context.Context, int64, int64) ([]model.BookingView, error) {
	return nil, nil
}
func (f *bookingRepoFake) ListByBusiness(context.Context, int64) ([]model.BookingView, error) {
	return nil, nil
}
func (f *bookingRepoFake) Cancel(context.Context, int64, int64, time.Time) error {
	f.cancelled = true
	return nil
}

type bookingUserRepoFake struct{ user *model.User }

func (f bookingUserRepoFake) GetByTelegramID(context.Context, int64) (*model.User, error) {
	return f.user, nil
}
func (bookingUserRepoFake) Create(context.Context, *model.User) (*model.User, error) { return nil, nil }
func (bookingUserRepoFake) Update(context.Context, *model.User) error                { return nil }

type bookingBusinessRepoFake struct{ business *model.Business }

func (f bookingBusinessRepoFake) GetByID(context.Context, int64) (*model.Business, error) {
	return f.business, nil
}
func (bookingBusinessRepoFake) Create(context.Context, *model.Business) (*model.Business, error) {
	return nil, nil
}

type bookingServiceRepoFake struct{ item *model.Service }

func (f bookingServiceRepoFake) GetByID(context.Context, int64) (*model.Service, error) {
	return f.item, nil
}
func (f bookingServiceRepoFake) ListByBusiness(context.Context, int64) ([]model.Service, error) {
	return []model.Service{*f.item}, nil
}

type bookingEmployeeRepoFake struct{ item *model.Employee }

func (f bookingEmployeeRepoFake) GetByID(context.Context, int64) (*model.Employee, error) {
	return f.item, nil
}
func (f bookingEmployeeRepoFake) ListByBusiness(context.Context, int64) ([]model.Employee, error) {
	return []model.Employee{*f.item}, nil
}

type bookingScheduleRepoFake struct {
	hours *model.WorkingHours
	busy  []model.TimeInterval
}

func (bookingScheduleRepoFake) UpsertWorkingHours(context.Context, *model.WorkingHours) (*model.WorkingHours, error) {
	return nil, nil
}
func (f bookingScheduleRepoFake) GetWorkingHours(context.Context, int64, int) (*model.WorkingHours, error) {
	return f.hours, nil
}
func (bookingScheduleRepoFake) CreateBlockedSlot(context.Context, *model.BlockedSlot) (*model.BlockedSlot, error) {
	return nil, nil
}
func (f bookingScheduleRepoFake) ListBusyIntervals(context.Context, int64, time.Time, time.Time) ([]model.TimeInterval, error) {
	return f.busy, nil
}

func newBookingTestService(repo *bookingRepoFake) *BookingService {
	business := &model.Business{ID: 1, Timezone: "Europe/Moscow", SlotIntervalMin: 30, MinBookingNoticeMin: 0, MaxBookingDays: 30}
	svc := bookingServiceRepoFake{item: &model.Service{ID: 2, BusinessID: 1, DurationMin: 60, Price: 1000, IsActive: true}}
	emp := bookingEmployeeRepoFake{item: &model.Employee{ID: 3, BusinessID: 1, IsActive: true}}
	schedule := NewScheduleService(bookingBusinessRepoFake{business}, svc, emp, bookingScheduleRepoFake{hours: &model.WorkingHours{Weekday: 4, StartMinute: 600, EndMinute: 1080}}, authorizerFake{})
	schedule.now = func() time.Time { return time.Date(2026, 10, 14, 10, 0, 0, 0, time.FixedZone("MSK", 3*3600)) }
	result := NewBookingService(repo, schedule, svc, emp, bookingUserRepoFake{user: &model.User{ID: 4, TelegramID: 5}}, bookingBusinessRepoFake{business})
	result.now = schedule.now
	return result
}

func TestBookingCreateValidatesSelectedSlot(t *testing.T) {
	t.Parallel()
	loc, _ := time.LoadLocation("Europe/Moscow")
	tests := []struct {
		name       string
		start      time.Time
		wantErr    error
		wantCreate bool
	}{{"available", time.Date(2026, 10, 15, 10, 0, 0, 0, loc), nil, true}, {"off grid", time.Date(2026, 10, 15, 10, 15, 0, 0, loc), apperr.ErrInvalidTime, false}, {"after hours", time.Date(2026, 10, 15, 18, 0, 0, 0, loc), apperr.ErrInvalidTime, false}}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &bookingRepoFake{}
			svc := newBookingTestService(repo)
			_, err := svc.Create(context.Background(), CreateBookingInput{BusinessID: 1, EmployeeID: 3, ServiceID: 2, TelegramID: 5, StartAt: tt.start})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error=%v want=%v", err, tt.wantErr)
			}
			if (repo.created != nil) != tt.wantCreate {
				t.Fatalf("created=%v want=%v", repo.created != nil, tt.wantCreate)
			}
		})
	}
}

func TestBookingCancelRules(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		booking model.Booking
		want    error
		cancel  bool
	}{{"future", model.Booking{ID: 1, UserID: 4, StartAt: now.Add(time.Hour), Status: model.BookingConfirmed}, nil, true}, {"other user", model.Booking{ID: 1, UserID: 9, StartAt: now.Add(time.Hour), Status: model.BookingConfirmed}, apperr.ErrForbidden, false}, {"started", model.Booking{ID: 1, UserID: 4, StartAt: now, Status: model.BookingConfirmed}, apperr.ErrBookingAlreadyStarted, false}, {"cancelled", model.Booking{ID: 1, UserID: 4, StartAt: now.Add(time.Hour), Status: model.BookingCancelled}, apperr.ErrBookingAlreadyCancelled, false}}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &bookingRepoFake{booking: &tt.booking}
			svc := newBookingTestService(repo)
			svc.now = func() time.Time { return now }
			err := svc.Cancel(context.Background(), 5, 1)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error=%v want=%v", err, tt.want)
			}
			if repo.cancelled != tt.cancel {
				t.Fatalf("cancelled=%v", repo.cancelled)
			}
		})
	}
}
