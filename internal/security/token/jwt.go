package token

import (
	"errors"
	"fmt"
	"time"

	"extrusion-quality-system/internal/domain"

	"github.com/golang-jwt/jwt/v5"
)

const minSecretLength = 32

type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now().UTC()
}

type JWTManager struct {
	secret []byte
	ttl    time.Duration
	clock  Clock
	issuer string
}

func NewJWTManager(secret string, ttl time.Duration, issuer string) (*JWTManager, error) {
	if len(secret) < minSecretLength {
		return nil, fmt.Errorf("jwt secret must contain at least %d characters", minSecretLength)
	}

	if ttl <= 0 {
		return nil, errors.New("jwt ttl must be positive")
	}

	if issuer == "" {
		issuer = "extrusion-quality-system"
	}

	return &JWTManager{
		secret: []byte(secret),
		ttl:    ttl,
		clock:  systemClock{},
		issuer: issuer,
	}, nil
}

func NewJWTManagerWithClock(
	secret string,
	ttl time.Duration,
	issuer string,
	clock Clock,
) (*JWTManager, error) {
	manager, err := NewJWTManager(secret, ttl, issuer)
	if err != nil {
		return nil, err
	}

	if clock != nil {
		manager.clock = clock
	}

	return manager, nil
}

func (m *JWTManager) Generate(user domain.User) (string, error) {
	now := m.clock.Now()
	expiresAt := now.Add(m.ttl)

	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subjectFromUserID(user.ID),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	signedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	rawToken, err := signedToken.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign jwt token: %w", err)
	}

	return rawToken, nil
}

func (m *JWTManager) Parse(rawToken string) (domain.AuthClaims, error) {
	claims := Claims{}

	parsedToken, err := jwt.ParseWithClaims(
		rawToken,
		&claims,
		func(parsedToken *jwt.Token) (any, error) {
			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return domain.AuthClaims{}, ErrExpiredToken
		}

		return domain.AuthClaims{}, ErrInvalidToken
	}

	if !parsedToken.Valid {
		return domain.AuthClaims{}, ErrInvalidToken
	}

	if claims.UserID <= 0 || claims.Username == "" || claims.Role == "" {
		return domain.AuthClaims{}, ErrInvalidToken
	}

	return claims.ToDomain(), nil
}
