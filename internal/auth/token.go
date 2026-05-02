package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"extrusion-quality-system/internal/domain"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

type Claims struct {
	UserID   domain.UserID `json:"userId"`
	Username string        `json:"username"`
	Role     domain.Role   `json:"role"`
	Expires  int64         `json:"exp"`
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
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		Expires:  time.Now().Add(m.ttl).Unix(),
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := m.sign(encodedPayload)

	return encodedPayload + "." + signature, nil
}

func (m *TokenManager) Parse(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return Claims{}, ErrInvalidToken
	}

	payload := parts[0]
	signature := parts[1]

	expectedSignature := m.sign(payload)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return Claims{}, ErrInvalidToken
	}

	rawPayload, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(rawPayload, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}

	if claims.Expires < time.Now().Unix() {
		return Claims{}, ErrExpiredToken
	}

	return claims, nil
}

func (m *TokenManager) sign(payload string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(payload))

	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (c Claims) Subject() string {
	return strconv.FormatInt(int64(c.UserID), 10)
}
