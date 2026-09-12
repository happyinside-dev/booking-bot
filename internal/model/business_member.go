package model

import "time"

type Role string

const (
	RoleOwner    Role = "OWNER"
	RoleAdmin    Role = "ADMIN"
	RoleEmployee Role = "EMPLOYEE"
)

type BusinessMember struct {
	ID         int64
	BusinessID int64
	UserID     int64
	Role       Role
	CreatedAt  time.Time
}
