package auth

import "extrusion-quality-system/internal/domain"

type LoginResult struct {
	Token string      `json:"token"`
	User  domain.User `json:"user"`
}
