package auth

import (
	"errors"
	"strconv"
	"time"

	"extrusion-quality-system/internal/domain"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

type Claims struct {
	UserID   domain.UserID   `json:"userId"`
	Username string          `json:"username"`
	Role     domain.UserRole `json:"role"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (m *TokenManager) Generate(user domain.User) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.ttl)

	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(int64(user.ID), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(m.secret)
}

func (m *TokenManager) Parse(rawToken string) (Claims, error) {
	claims := Claims{}

	token, err := jwt.ParseWithClaims(
		rawToken,
		&claims,
		func(token *jwt.Token) (any, error) {
			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return Claims{}, ErrExpiredToken
		}

		return Claims{}, ErrInvalidToken
	}

	if !token.Valid {
		return Claims{}, ErrInvalidToken
	}

	if claims.UserID <= 0 || claims.Username == "" || claims.Role == "" {
		return Claims{}, ErrInvalidToken
	}

	return claims, nil
}
