package token

import (
	"errors"
	"testing"
	"time"

	"extrusion-quality-system/internal/domain"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func TestJWTManagerGenerateAndParse(t *testing.T) {
	now := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)

	manager, err := NewJWTManagerWithClock(
		"test-secret-with-more-than-32-characters",
		time.Hour,
		"extrusion-quality-system",
		fixedClock{now: now},
	)
	if err != nil {
		t.Fatalf("NewJWTManagerWithClock returned error: %v", err)
	}

	user := domain.User{
		ID:       2,
		Username: "maria.sokolova",
		Role:     domain.UserRoleTechnologist,
		IsActive: true,
	}

	rawToken, err := manager.Generate(user)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	if rawToken == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := manager.Parse(rawToken)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if claims.UserID != user.ID {
		t.Fatalf("UserID = %d, want %d", claims.UserID, user.ID)
	}

	if claims.Username != user.Username {
		t.Fatalf("Username = %q, want %q", claims.Username, user.Username)
	}

	if claims.Role != user.Role {
		t.Fatalf("Role = %q, want %q", claims.Role, user.Role)
	}
}

func TestJWTManagerRejectsShortSecret(t *testing.T) {
	_, err := NewJWTManager(
		"short",
		time.Hour,
		"extrusion-quality-system",
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestJWTManagerRejectsNonPositiveTTL(t *testing.T) {
	_, err := NewJWTManager(
		"test-secret-with-more-than-32-characters",
		0,
		"extrusion-quality-system",
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestJWTManagerParseInvalidToken(t *testing.T) {
	manager, err := NewJWTManager(
		"test-secret-with-more-than-32-characters",
		time.Hour,
		"extrusion-quality-system",
	)
	if err != nil {
		t.Fatalf("NewJWTManager returned error: %v", err)
	}

	_, err = manager.Parse("not-a-valid-token")

	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("error = %v, want ErrInvalidToken", err)
	}
}

func TestJWTManagerParseExpiredToken(t *testing.T) {
	now := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)

	manager, err := NewJWTManagerWithClock(
		"test-secret-with-more-than-32-characters",
		time.Nanosecond,
		"extrusion-quality-system",
		fixedClock{now: now},
	)
	if err != nil {
		t.Fatalf("NewJWTManagerWithClock returned error: %v", err)
	}

	user := domain.User{
		ID:       2,
		Username: "maria.sokolova",
		Role:     domain.UserRoleTechnologist,
		IsActive: true,
	}

	rawToken, err := manager.Generate(user)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	time.Sleep(time.Millisecond)

	_, err = manager.Parse(rawToken)

	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("error = %v, want ErrExpiredToken", err)
	}
}
