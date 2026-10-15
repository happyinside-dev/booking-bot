package service

import (
	"context"
	"errors"
	"testing"

	"github.com/happyinside-dev/booking-bot/internal/apperr"
	"github.com/happyinside-dev/booking-bot/internal/model"
)

type catalogRepoFake struct {
	item    *model.Service
	created bool
	updated bool
}

func (f *catalogRepoFake) Create(_ context.Context, item *model.Service) (*model.Service, error) {
	f.created = true
	item.ID = 1
	item.IsActive = true
	f.item = item
	return item, nil
}
func (f *catalogRepoFake) GetByID(_ context.Context, _ int64) (*model.Service, error) {
	if f.item == nil {
		return nil, apperr.ErrNotFound
	}
	copy := *f.item
	return &copy, nil
}
func (f *catalogRepoFake) ListByBusiness(_ context.Context, _ int64) ([]model.Service, error) {
	if f.item == nil {
		return nil, nil
	}
	return []model.Service{*f.item}, nil
}
func (f *catalogRepoFake) Update(_ context.Context, item *model.Service) error {
	f.updated = true
	f.item = item
	return nil
}

type authorizerFake struct{ err error }

func (f authorizerFake) HasRole(context.Context, int64, int64, ...model.Role) error { return f.err }

func TestCatalogCreateAuthorizationAndValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		authErr    error
		input      CreateServiceInput
		wantErr    error
		wantCreate bool
	}{
		{name: "owner creates valid service", input: CreateServiceInput{ActorUserID: 1, BusinessID: 2, Name: "Стрижка", DurationMin: 60, Price: 1500}, wantCreate: true},
		{name: "forbidden actor", authErr: apperr.ErrForbidden, input: CreateServiceInput{ActorUserID: 1, BusinessID: 2, Name: "Стрижка", DurationMin: 60}, wantErr: apperr.ErrForbidden},
		{name: "invalid duration", input: CreateServiceInput{ActorUserID: 1, BusinessID: 2, Name: "Стрижка", DurationMin: 0}, wantErr: apperr.ErrInvalidInput},
		{name: "negative price", input: CreateServiceInput{ActorUserID: 1, BusinessID: 2, Name: "Стрижка", DurationMin: 30, Price: -1}, wantErr: apperr.ErrInvalidInput},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &catalogRepoFake{}
			svc := NewCatalogService(repo, authorizerFake{err: tt.authErr})
			_, err := svc.Create(context.Background(), tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if repo.created != tt.wantCreate {
				t.Fatalf("created = %v, want %v", repo.created, tt.wantCreate)
			}
		})
	}
}

func TestCatalogRejectsCrossBusinessUpdate(t *testing.T) {
	t.Parallel()
	repo := &catalogRepoFake{item: &model.Service{ID: 7, BusinessID: 99, Name: "Other", DurationMin: 30, IsActive: true}}
	svc := NewCatalogService(repo, authorizerFake{})
	_, err := svc.Update(context.Background(), UpdateServiceInput{ActorUserID: 1, BusinessID: 2, ServiceID: 7, Name: "Changed", DurationMin: 30})
	if !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("error = %v, want forbidden", err)
	}
	if repo.updated {
		t.Fatal("cross-business service was updated")
	}
}
