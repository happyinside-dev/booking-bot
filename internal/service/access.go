package service

import (
	"context"

	"github.com/happyinside-dev/booking-bot/internal/model"
)

// RoleAuthorizer is the centralized authorization boundary used by every
// administrative service. BusinessService implements it.
type RoleAuthorizer interface {
	HasRole(ctx context.Context, businessID, userID int64, allowed ...model.Role) error
}

func authorizeAdmin(ctx context.Context, auth RoleAuthorizer, businessID, actorUserID int64) error {
	return auth.HasRole(ctx, businessID, actorUserID, model.RoleOwner, model.RoleAdmin)
}
