package auth

import "extrusion-quality-system/internal/domain"

type LoginInput struct {
	Username string
	Password string
}

type ChangePasswordInput struct {
	UserID      domain.UserID
	OldPassword string
	NewPassword string
}
