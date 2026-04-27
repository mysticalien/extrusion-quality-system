package domain

import "time"

type UserID int64

type Role string

const (
	RoleOperator     Role = "operator"
	RoleTechnologist Role = "technologist"
	RoleAdmin        Role = "admin"
)

// User represents an application user with role-based access permissions.
type User struct {
	ID           UserID    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	IsActive     bool      `json:"isActive"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (u User) CanAcknowledgeAlerts() bool {
	return u.Role == RoleOperator || u.Role == RoleTechnologist || u.Role == RoleAdmin
}

func (u User) CanUpdateSetpoints() bool {
	return u.Role == RoleTechnologist || u.Role == RoleAdmin
}

func (u User) CanManageUsers() bool {
	return u.Role == RoleAdmin
}
