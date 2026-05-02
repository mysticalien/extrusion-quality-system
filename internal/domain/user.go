package domain

import (
	"errors"
	"strings"
	"time"
)

// UserID identifies an application user.
type UserID int64

// UserRole defines a user access level in the system.
type UserRole string

const (
	UserRoleOperator     UserRole = "operator"
	UserRoleTechnologist UserRole = "technologist"
	UserRoleAdmin        UserRole = "admin"
)

// User represents an application user with role-based access permissions.
type User struct {
	ID       UserID `json:"id"`
	Username string `json:"username"`

	// PasswordHash is excluded from JSON responses for security reasons.
	PasswordHash string `json:"-"`

	Role     UserRole `json:"role"`
	IsActive bool     `json:"isActive"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type UserCreate struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Role     UserRole `json:"role"`
	IsActive bool     `json:"isActive"`
}

type UserRoleUpdate struct {
	Role UserRole `json:"role"`
}

type UserPasswordUpdate struct {
	Password string `json:"password"`
}

type UserChangePassword struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// CanAcknowledgeAlerts reports whether the user can acknowledge alert events.
func (u User) CanAcknowledgeAlerts() bool {
	return u.Role == UserRoleOperator || u.Role == UserRoleTechnologist || u.Role == UserRoleAdmin
}

// CanUpdateSetpoints reports whether the user can update technological setpoints.
func (u User) CanUpdateSetpoints() bool {
	return u.Role == UserRoleTechnologist || u.Role == UserRoleAdmin
}

// CanManageUsers reports whether the user can manage application users.
func (u User) CanManageUsers() bool {
	return u.Role == UserRoleAdmin
}

func IsValidUserRole(role UserRole) bool {
	switch role {
	case UserRoleOperator, UserRoleTechnologist, UserRoleAdmin:
		return true
	default:
		return false
	}
}

func ValidateUserCreate(input UserCreate) error {
	if strings.TrimSpace(input.Username) == "" {
		return errors.New("username is required")
	}

	if len(strings.TrimSpace(input.Username)) < 3 {
		return errors.New("username must contain at least 3 characters")
	}

	if len(input.Password) < 12 {
		return errors.New("password must contain at least 12 characters")
	}

	if !IsValidUserRole(input.Role) {
		return errors.New("invalid user role")
	}

	return nil
}

func ValidateUserRoleUpdate(input UserRoleUpdate) error {
	if !IsValidUserRole(input.Role) {
		return errors.New("invalid user role")
	}

	return nil
}

func ValidateUserPasswordUpdate(input UserPasswordUpdate) error {
	if len(input.Password) < 12 {
		return errors.New("password must contain at least 12 characters")
	}

	return nil
}

func ValidateUserChangePassword(input UserChangePassword) error {
	if strings.TrimSpace(input.OldPassword) == "" {
		return errors.New("old password is required")
	}

	if len(input.NewPassword) < 12 {
		return errors.New("new password must contain at least 12 characters")
	}

	if input.OldPassword == input.NewPassword {
		return errors.New("new password must be different from old password")
	}

	return nil
}
