// Package apperr defines domain/service-level sentinel errors shared across
// the service and repository layers. Handlers translate these into
// human-readable Telegram messages instead of leaking internal details.
//
// This file grows incrementally: only errors actually used by an
// implemented stage are declared here.
package apperr

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrForbidden       = errors.New("forbidden")
	ErrInvalidSchedule = errors.New("invalid schedule")
	ErrInvalidTimezone = errors.New("invalid timezone")
	ErrInvalidInput    = errors.New("invalid input")

	// Ошибки бронирования (понадобятся на следующих этапах)
	ErrBookingConflict         = errors.New("booking conflict")
	ErrInvalidTime             = errors.New("invalid time")
	ErrServiceInactive         = errors.New("service inactive")
	ErrEmployeeInactive        = errors.New("employee inactive")
	ErrBookingAlreadyCancelled = errors.New("booking already cancelled")
	ErrBookingAlreadyStarted   = errors.New("booking already started")
)
