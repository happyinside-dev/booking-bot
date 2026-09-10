package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/yourname/booking-bot/internal/apperr"
	"github.com/yourname/booking-bot/internal/model"
)

// userRepository is the subset of repository.UserRepository this service
// depends on, declared here so the service can be unit-tested with a fake.
type userRepository interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*model.User, error)
	Create(ctx context.Context, u *model.User) (*model.User, error)
	Update(ctx context.Context, u *model.User) error
}

type UserService struct {
	repo userRepository
	log  *slog.Logger
}

func NewUserService(repo userRepository, log *slog.Logger) *UserService {
	return &UserService{repo: repo, log: log}
}

// RegisterUserInput carries the subset of Telegram profile data relevant to
// our users table. It exists so the service layer has no import dependency
// on any Telegram library type.
type RegisterUserInput struct {
	TelegramID int64
	Username   string // empty string if Telegram user has none
	FirstName  string
	LastName   string
}

// RegisterOrUpdate is called on every /start. If the Telegram user is new,
// it creates a row. If the user already exists, it refreshes profile fields
// that may have changed in Telegram (username, first/last name) without
// touching phone (which is only ever set by our own flows).
func (s *UserService) RegisterOrUpdate(ctx context.Context, in RegisterUserInput) (*model.User, error) {
	existing, err := s.repo.GetByTelegramID(ctx, in.TelegramID)
	if err != nil {
		if !errors.Is(err, apperr.ErrNotFound) {
			return nil, fmt.Errorf("service: lookup user: %w", err)
		}

		u := &model.User{
			TelegramID: in.TelegramID,
			Username:   nilIfEmpty(in.Username),
			FirstName:  nilIfEmpty(in.FirstName),
			LastName:   nilIfEmpty(in.LastName),
		}

		created, err := s.repo.Create(ctx, u)
		if err != nil {
			return nil, fmt.Errorf("service: create user: %w", err)
		}

		s.log.Info("registered new user", slog.Int64("telegram_id", in.TelegramID), slog.Int64("user_id", created.ID))
		return created, nil
	}

	changed := false
	if !strPtrEqual(existing.Username, in.Username) {
		existing.Username = nilIfEmpty(in.Username)
		changed = true
	}
	if !strPtrEqual(existing.FirstName, in.FirstName) {
		existing.FirstName = nilIfEmpty(in.FirstName)
		changed = true
	}
	if !strPtrEqual(existing.LastName, in.LastName) {
		existing.LastName = nilIfEmpty(in.LastName)
		changed = true
	}

	if changed {
		if err := s.repo.Update(ctx, existing); err != nil {
			return nil, fmt.Errorf("service: update user: %w", err)
		}
		s.log.Info("updated user profile", slog.Int64("telegram_id", in.TelegramID), slog.Int64("user_id", existing.ID))
	}

	return existing, nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func strPtrEqual(p *string, s string) bool {
	if p == nil {
		return s == ""
	}
	return *p == s
}
