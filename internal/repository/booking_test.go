package repository

import (
	"errors"
	"testing"

	"github.com/happyinside-dev/booking-bot/internal/apperr"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapBookingWriteError(t *testing.T) {
	t.Parallel()
	conflict := &pgconn.PgError{Code: "23P01"}
	if err := mapBookingWriteError("insert", conflict); !errors.Is(err, apperr.ErrBookingConflict) {
		t.Fatalf("error = %v", err)
	}
	original := errors.New("connection lost")
	if err := mapBookingWriteError("insert", original); !errors.Is(err, original) {
		t.Fatalf("error = %v", err)
	}
}
