package storage

import (
	"sync"
	"time"

	"extrusion-quality-system/internal/domain"
)

// MemoryAlertStore stores alert events in memory.
// It is used only for the first prototype and tests.
type MemoryAlertStore struct {
	mu     sync.RWMutex
	nextID domain.AlertID
	alerts []domain.AlertEvent
}

// NewMemoryAlertStore creates an empty in-memory alert store.
func NewMemoryAlertStore() *MemoryAlertStore {
	return &MemoryAlertStore{
		nextID: 1,
	}
}

// Create stores an alert event and assigns an in-memory ID.
func (s *MemoryAlertStore) Create(alert domain.AlertEvent) (domain.AlertEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	alert.ID = s.nextID
	s.nextID++

	if alert.Status == "" {
		alert.Status = domain.AlertStatusActive
	}

	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = time.Now().UTC()
	}

	s.alerts = append(s.alerts, alert)

	return alert, nil
}

// All returns a copy of all stored alert events.
func (s *MemoryAlertStore) All() ([]domain.AlertEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]domain.AlertEvent, len(s.alerts))
	copy(result, s.alerts)

	return result, nil
}

// Active returns all alerts that still affect the current process state.
func (s *MemoryAlertStore) Active() ([]domain.AlertEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]domain.AlertEvent, 0)

	for _, alert := range s.alerts {
		if alert.Status == domain.AlertStatusActive || alert.Status == domain.AlertStatusAcknowledged {
			result = append(result, alert)
		}
	}

	return result, nil
}

// Acknowledge marks an alert as acknowledged.
func (s *MemoryAlertStore) Acknowledge(id domain.AlertID, userID *domain.UserID) (domain.AlertEvent, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.alerts {
		if s.alerts[i].ID == id {
			s.alerts[i].Acknowledge(userID, time.Now().UTC())
			return s.alerts[i], true, nil
		}
	}

	return domain.AlertEvent{}, false, nil
}

// Resolve marks an alert as resolved.
func (s *MemoryAlertStore) Resolve(id domain.AlertID) (domain.AlertEvent, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.alerts {
		if s.alerts[i].ID == id {
			s.alerts[i].Resolve(time.Now().UTC())
			return s.alerts[i], true, nil
		}
	}

	return domain.AlertEvent{}, false, nil
}
