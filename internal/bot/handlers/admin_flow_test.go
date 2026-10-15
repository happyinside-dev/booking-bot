package handlers

import (
	"errors"
	"testing"

	"github.com/happyinside-dev/booking-bot/internal/apperr"
)

func TestAdminInputParsing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		run     func() error
		wantErr bool
	}{
		{name: "clock", run: func() error {
			value, err := parseClock("09:30")
			if value != 570 {
				t.Fatalf("minutes = %d", value)
			}
			return err
		}},
		{name: "invalid clock", run: func() error { _, err := parseClock("25:00"); return err }, wantErr: true},
		{name: "comma price", run: func() error {
			value, err := nonNegativeFloat("1500,50")
			if value != 1500.5 {
				t.Fatalf("price = %v", value)
			}
			return err
		}},
		{name: "negative price", run: func() error { _, err := nonNegativeFloat("-1"); return err }, wantErr: true},
		{name: "activate", run: func() error {
			active, err := parseActive("вкл")
			if !active {
				t.Fatal("not active")
			}
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if tt.wantErr != errors.Is(err, apperr.ErrInvalidInput) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
