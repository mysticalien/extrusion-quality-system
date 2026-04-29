package domain

import "time"

// UserID identifies an application user.
type UserID int64

// Role defines a user access level in the system.
type Role string

const (
	RoleOperator     Role = "operator"
	RoleTechnologist Role = "technologist"
	RoleAdmin        Role = "admin"
)

// User represents an application user with role-based access permissions.
type User struct {
	ID       UserID `json:"id"`
	Username string `json:"username"`

	// PasswordHash is excluded from JSON responses for security reasons.
	PasswordHash string `json:"-"`

	Role     Role `json:"role"`
	IsActive bool `json:"isActive"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CanAcknowledgeAlerts reports whether the user can acknowledge alert events.
func (u User) CanAcknowledgeAlerts() bool {
	return u.Role == RoleOperator || u.Role == RoleTechnologist || u.Role == RoleAdmin
}

// CanUpdateSetpoints reports whether the user can update technological setpoints.
func (u User) CanUpdateSetpoints() bool {
	return u.Role == RoleTechnologist || u.Role == RoleAdmin
}

// CanManageUsers reports whether the user can manage application users.
func (u User) CanManageUsers() bool {
	return u.Role == RoleAdmin
}
