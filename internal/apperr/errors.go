// Package apperr defines domain/service-level sentinel errors shared across
// the service and repository layers. Handlers translate these into
// human-readable Telegram messages instead of leaking internal details.
//
// This file grows incrementally: only errors actually used by an
// implemented stage are declared here.
package apperr

import "errors"

var (
	// ErrNotFound is returned when a requested entity does not exist.
	ErrNotFound = errors.New("not found")
)
