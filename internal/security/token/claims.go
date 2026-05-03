package token

import (
	"strconv"

	"extrusion-quality-system/internal/domain"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   domain.UserID   `json:"userId"`
	Username string          `json:"username"`
	Role     domain.UserRole `json:"role"`
	jwt.RegisteredClaims
}

func (c Claims) ToDomain() domain.AuthClaims {
	return domain.AuthClaims{
		UserID:   c.UserID,
		Username: c.Username,
		Role:     c.Role,
	}
}

func subjectFromUserID(userID domain.UserID) string {
	return strconv.FormatInt(int64(userID), 10)
}
