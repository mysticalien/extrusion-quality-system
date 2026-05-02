package storage

import (
	"context"
	"sync"
	"time"

	"extrusion-quality-system/internal/domain"
)

type MemoryUserRepository struct {
	mu     sync.RWMutex
	users  map[domain.UserID]domain.User
	byName map[string]domain.UserID
}

func NewMemoryUserRepository(users []domain.User) *MemoryUserRepository {
	repository := &MemoryUserRepository{
		users:  make(map[domain.UserID]domain.User),
		byName: make(map[string]domain.UserID),
	}

	for _, user := range users {
		if user.CreatedAt.IsZero() {
			user.CreatedAt = time.Now().UTC()
		}

		if user.UpdatedAt.IsZero() {
			user.UpdatedAt = time.Now().UTC()
		}

		repository.users[user.ID] = user
		repository.byName[user.Username] = user.ID
	}

	return repository
}

func (r *MemoryUserRepository) FindByUsername(
	_ context.Context,
	username string,
) (domain.User, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byName[username]
	if !ok {
		return domain.User{}, false, nil
	}

	user, ok := r.users[id]

	return user, ok, nil
}

func (r *MemoryUserRepository) FindByID(
	_ context.Context,
	id domain.UserID,
) (domain.User, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[id]

	return user, ok, nil
}
