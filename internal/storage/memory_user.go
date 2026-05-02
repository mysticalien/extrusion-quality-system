package storage

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"extrusion-quality-system/internal/domain"
)

var ErrMemoryUserAlreadyExists = errors.New("user already exists")

type MemoryUserRepository struct {
	mu     sync.RWMutex
	nextID domain.UserID
	users  map[domain.UserID]domain.User
	byName map[string]domain.UserID
}

func NewMemoryUserRepository(users []domain.User) *MemoryUserRepository {
	repository := &MemoryUserRepository{
		nextID: 1,
		users:  make(map[domain.UserID]domain.User),
		byName: make(map[string]domain.UserID),
	}

	for _, user := range users {
		if user.ID == 0 {
			user.ID = repository.nextID
			repository.nextID++
		}

		if user.CreatedAt.IsZero() {
			user.CreatedAt = time.Now().UTC()
		}

		if user.UpdatedAt.IsZero() {
			user.UpdatedAt = user.CreatedAt
		}

		repository.users[user.ID] = user
		repository.byName[user.Username] = user.ID

		if user.ID >= repository.nextID {
			repository.nextID = user.ID + 1
		}
	}

	return repository
}

func (r *MemoryUserRepository) All(_ context.Context) ([]domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]domain.User, 0, len(r.users))

	for _, user := range r.users {
		users = append(users, user)
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].ID < users[j].ID
	})

	return users, nil
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

func (r *MemoryUserRepository) Create(
	_ context.Context,
	user domain.User,
) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byName[user.Username]; exists {
		return domain.User{}, ErrMemoryUserAlreadyExists
	}

	now := time.Now().UTC()

	user.ID = r.nextID
	user.CreatedAt = now
	user.UpdatedAt = now

	r.nextID++
	r.users[user.ID] = user
	r.byName[user.Username] = user.ID

	return user, nil
}

func (r *MemoryUserRepository) UpdateRole(
	_ context.Context,
	id domain.UserID,
	role domain.UserRole,
) (domain.User, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.users[id]
	if !ok {
		return domain.User{}, false, nil
	}

	user.Role = role
	user.UpdatedAt = time.Now().UTC()

	r.users[id] = user

	return user, true, nil
}

func (r *MemoryUserRepository) UpdatePassword(
	_ context.Context,
	id domain.UserID,
	passwordHash string,
) (domain.User, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.users[id]
	if !ok {
		return domain.User{}, false, nil
	}

	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now().UTC()

	r.users[id] = user

	return user, true, nil
}

func (r *MemoryUserRepository) SetActive(
	_ context.Context,
	id domain.UserID,
	isActive bool,
) (domain.User, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.users[id]
	if !ok {
		return domain.User{}, false, nil
	}

	user.IsActive = isActive
	user.UpdatedAt = time.Now().UTC()

	r.users[id] = user

	return user, true, nil
}
