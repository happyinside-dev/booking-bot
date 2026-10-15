package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/happyinside-dev/booking-bot/internal/apperr"
	"github.com/happyinside-dev/booking-bot/internal/model"
)

type scheduleBusinessRepository interface {
	GetByID(ctx context.Context, id int64) (*model.Business, error)
}

type scheduleServiceRepository interface {
	GetByID(ctx context.Context, id int64) (*model.Service, error)
}

type scheduleEmployeeRepository interface {
	GetByID(ctx context.Context, id int64) (*model.Employee, error)
}

type scheduleRepository interface {
	UpsertWorkingHours(ctx context.Context, h *model.WorkingHours) (*model.WorkingHours, error)
	GetWorkingHours(ctx context.Context, employeeID int64, weekday int) (*model.WorkingHours, error)
	CreateBlockedSlot(ctx context.Context, slot *model.BlockedSlot) (*model.BlockedSlot, error)
	ListBusyIntervals(ctx context.Context, employeeID int64, from, to time.Time) ([]model.TimeInterval, error)
}

type ScheduleService struct {
	businesses scheduleBusinessRepository
	services   scheduleServiceRepository
	employees  scheduleEmployeeRepository
	schedules  scheduleRepository
	auth       RoleAuthorizer
	now        func() time.Time
}

func NewScheduleService(
	businesses scheduleBusinessRepository,
	services scheduleServiceRepository,
	employees scheduleEmployeeRepository,
	schedules scheduleRepository,
	auth RoleAuthorizer,
) *ScheduleService {
	return &ScheduleService{businesses: businesses, services: services, employees: employees, schedules: schedules, auth: auth, now: time.Now}
}

type SetWorkingHoursInput struct {
	ActorUserID int64
	BusinessID  int64
	EmployeeID  int64
	Weekday     int
	StartMinute int
	EndMinute   int
	IsDayOff    bool
}

func (s *ScheduleService) SetWorkingHours(ctx context.Context, in SetWorkingHoursInput) (*model.WorkingHours, error) {
	if err := authorizeAdmin(ctx, s.auth, in.BusinessID, in.ActorUserID); err != nil {
		return nil, err
	}
	if in.Weekday < 1 || in.Weekday > 7 || in.StartMinute < 0 || in.StartMinute >= 24*60 ||
		in.EndMinute < 0 || in.EndMinute > 24*60 || (!in.IsDayOff && in.StartMinute >= in.EndMinute) {
		return nil, apperr.ErrInvalidSchedule
	}
	employee, err := s.employees.GetByID(ctx, in.EmployeeID)
	if err != nil {
		return nil, fmt.Errorf("service: get employee: %w", err)
	}
	if employee.BusinessID != in.BusinessID {
		return nil, apperr.ErrForbidden
	}
	return s.schedules.UpsertWorkingHours(ctx, &model.WorkingHours{
		EmployeeID: in.EmployeeID, Weekday: in.Weekday, StartMinute: in.StartMinute,
		EndMinute: in.EndMinute, IsDayOff: in.IsDayOff,
	})
}

type CreateBlockedSlotInput struct {
	ActorUserID int64
	BusinessID  int64
	EmployeeID  int64
	StartAt     time.Time
	EndAt       time.Time
	Reason      string
}

func (s *ScheduleService) CreateBlockedSlot(ctx context.Context, in CreateBlockedSlotInput) (*model.BlockedSlot, error) {
	if err := authorizeAdmin(ctx, s.auth, in.BusinessID, in.ActorUserID); err != nil {
		return nil, err
	}
	if in.StartAt.IsZero() || !in.StartAt.Before(in.EndAt) {
		return nil, apperr.ErrInvalidSchedule
	}
	employee, err := s.employees.GetByID(ctx, in.EmployeeID)
	if err != nil {
		return nil, fmt.Errorf("service: get employee: %w", err)
	}
	if employee.BusinessID != in.BusinessID {
		return nil, apperr.ErrForbidden
	}
	return s.schedules.CreateBlockedSlot(ctx, &model.BlockedSlot{
		EmployeeID: in.EmployeeID, StartAt: in.StartAt.UTC(), EndAt: in.EndAt.UTC(), Reason: in.Reason,
	})
}

type AvailableSlotsRequest struct {
	EmployeeID int64
	ServiceID  int64
	Date       time.Time
}

func (s *ScheduleService) GetAvailableSlots(ctx context.Context, req AvailableSlotsRequest) ([]time.Time, error) {
	employee, err := s.employees.GetByID(ctx, req.EmployeeID)
	if err != nil {
		return nil, fmt.Errorf("service: get employee: %w", err)
	}
	if !employee.IsActive {
		return nil, apperr.ErrEmployeeInactive
	}

	service, err := s.services.GetByID(ctx, req.ServiceID)
	if err != nil {
		return nil, fmt.Errorf("service: get service: %w", err)
	}
	if !service.IsActive {
		return nil, apperr.ErrServiceInactive
	}
	if service.BusinessID != employee.BusinessID {
		return nil, apperr.ErrForbidden
	}

	business, err := s.businesses.GetByID(ctx, employee.BusinessID)
	if err != nil {
		return nil, fmt.Errorf("service: get business: %w", err)
	}
	loc, err := time.LoadLocation(business.Timezone)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", apperr.ErrInvalidTimezone, business.Timezone)
	}
	if business.SlotIntervalMin <= 0 || business.MinBookingNoticeMin < 0 || business.MaxBookingDays < 0 || service.DurationMin <= 0 {
		return nil, apperr.ErrInvalidSchedule
	}

	year, month, day := req.Date.Date()
	date := time.Date(year, month, day, 0, 0, 0, 0, loc)
	now := s.now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	if date.Before(today) || date.After(today.AddDate(0, 0, business.MaxBookingDays)) {
		return []time.Time{}, nil
	}

	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	hours, err := s.schedules.GetWorkingHours(ctx, req.EmployeeID, weekday)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return []time.Time{}, nil
		}
		return nil, fmt.Errorf("service: get working hours: %w", err)
	}
	if hours.IsDayOff {
		return []time.Time{}, nil
	}
	if hours.StartMinute < 0 || hours.EndMinute > 24*60 || hours.StartMinute >= hours.EndMinute {
		return nil, apperr.ErrInvalidSchedule
	}

	workStart := atLocalMinute(date, hours.StartMinute, loc)
	workEnd := atLocalMinute(date, hours.EndMinute, loc)
	busy, err := s.schedules.ListBusyIntervals(ctx, req.EmployeeID, workStart.UTC(), workEnd.UTC())
	if err != nil {
		return nil, fmt.Errorf("service: list busy intervals: %w", err)
	}

	return ComputeAvailableSlots(
		workStart, workEnd,
		time.Duration(service.DurationMin)*time.Minute,
		time.Duration(business.SlotIntervalMin)*time.Minute,
		now.Add(time.Duration(business.MinBookingNoticeMin)*time.Minute),
		busy,
	), nil
}

func atLocalMinute(date time.Time, minute int, loc *time.Location) time.Time {
	extraDay, minuteOfDay := minute/(24*60), minute%(24*60)
	return time.Date(date.Year(), date.Month(), date.Day()+extraDay, minuteOfDay/60, minuteOfDay%60, 0, 0, loc)
}

// ComputeAvailableSlots is the pure, table-testable core of slot generation.
// It uses half-open intervals [start,end), so adjacent appointments are valid.
func ComputeAvailableSlots(workStart, workEnd time.Time, duration, interval time.Duration, minStart time.Time, busy []model.TimeInterval) []time.Time {
	if !workStart.Before(workEnd) || duration <= 0 || interval <= 0 {
		return []time.Time{}
	}
	busy = append([]model.TimeInterval(nil), busy...)
	sort.Slice(busy, func(i, j int) bool {
		if busy[i].Start.Equal(busy[j].Start) {
			return busy[i].End.Before(busy[j].End)
		}
		return busy[i].Start.Before(busy[j].Start)
	})
	result := make([]time.Time, 0)
	busyIndex := 0
	for start := workStart; !start.Add(duration).After(workEnd); start = start.Add(interval) {
		end := start.Add(duration)
		if start.Before(minStart) {
			continue
		}
		for busyIndex < len(busy) && !busy[busyIndex].End.After(start) {
			busyIndex++
		}
		conflict := false
		for i := busyIndex; i < len(busy) && busy[i].Start.Before(end); i++ {
			if start.Before(busy[i].End) && end.After(busy[i].Start) {
				conflict = true
				break
			}
		}
		if !conflict {
			result = append(result, start.UTC())
		}
	}
	return result
}
