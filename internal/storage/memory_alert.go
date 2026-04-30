package storage

import (
	"context"
	"sync"
	"time"

	"extrusion-quality-system/internal/domain"
)

// MemoryAlertRepository stores alert events in memory.
// It is used for tests and early prototypes.
type MemoryAlertRepository struct {
	mu     sync.RWMutex
	nextID domain.AlertID
	alerts []domain.AlertEvent
}

// NewMemoryAlertRepository creates an empty in-memory alert repository.
func NewMemoryAlertRepository() *MemoryAlertRepository {
	return &MemoryAlertRepository{
		nextID: 1,
	}
}

// Create stores an alert event and assigns an in-memory ID.
func (r *MemoryAlertRepository) Create(
	_ context.Context,
	alert domain.AlertEvent,
) (domain.AlertEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	alert.ID = r.nextID
	r.nextID++

	if alert.Status == "" {
		alert.Status = domain.AlertStatusActive
	}

	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = time.Now().UTC()
	}

	r.alerts = append(r.alerts, alert)

	return alert, nil
}

// All returns a copy of all stored alert events.
func (r *MemoryAlertRepository) All(_ context.Context) ([]domain.AlertEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.AlertEvent, len(r.alerts))
	copy(result, r.alerts)

	return result, nil
}

// Active returns alerts that still affect the current process state.
func (r *MemoryAlertRepository) Active(_ context.Context) ([]domain.AlertEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.AlertEvent, 0)

	for _, alert := range r.alerts {
		if alert.Status == domain.AlertStatusActive || alert.Status == domain.AlertStatusAcknowledged {
			result = append(result, alert)
		}
	}

	return result, nil
}

// Acknowledge marks an alert as acknowledged.
func (r *MemoryAlertRepository) Acknowledge(
	_ context.Context,
	id domain.AlertID,
	userID *domain.UserID,
) (domain.AlertEvent, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.alerts {
		if r.alerts[i].ID == id {
			r.alerts[i].Acknowledge(userID, time.Now().UTC())
			return r.alerts[i], true, nil
		}
	}

	return domain.AlertEvent{}, false, nil
}

// Resolve marks an alert as resolved.
func (r *MemoryAlertRepository) Resolve(
	_ context.Context,
	id domain.AlertID,
) (domain.AlertEvent, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.alerts {
		if r.alerts[i].ID == id {
			r.alerts[i].Resolve(time.Now().UTC())
			return r.alerts[i], true, nil
		}
	}

	return domain.AlertEvent{}, false, nil
}
