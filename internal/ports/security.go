package ports

import "extrusion-quality-system/internal/domain"

type PasswordHasher interface {
	Hash(password string) (string, error)
	Check(password string, passwordHash string) bool
}

type TokenManager interface {
	Generate(user domain.User) (string, error)
	Parse(rawToken string) (domain.AuthClaims, error)
}
