package password

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type BcryptHasher struct {
	cost int
}

func NewBcryptHasher(cost int) *BcryptHasher {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}

	return &BcryptHasher{
		cost: cost,
	}
}

func (h *BcryptHasher) Hash(rawPassword string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(rawPassword), h.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	return string(hash), nil
}

func (h *BcryptHasher) Check(rawPassword string, passwordHash string) bool {
	err := bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(rawPassword),
	)

	return err == nil
}
