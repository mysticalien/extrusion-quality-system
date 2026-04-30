package storage

import (
	"context"
	"sort"
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

	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID > result[j].ID
		}

		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result, nil
}

// Active returns the latest open alert for each parameter.
// Open alert means active or acknowledged.
func (r *MemoryAlertRepository) Active(_ context.Context) ([]domain.AlertEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	latestByParameter := make(map[domain.ParameterType]domain.AlertEvent)

	for _, alert := range r.alerts {
		if !isOpenAlert(alert) {
			continue
		}

		current, exists := latestByParameter[alert.ParameterType]
		if !exists || alert.CreatedAt.After(current.CreatedAt) ||
			alert.CreatedAt.Equal(current.CreatedAt) && alert.ID > current.ID {
			latestByParameter[alert.ParameterType] = alert
		}
	}

	result := make([]domain.AlertEvent, 0, len(latestByParameter))
	for _, alert := range latestByParameter {
		result = append(result, alert)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID > result[j].ID
		}

		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result, nil
}

// FindOpenByParameter returns the latest active or acknowledged alert for a parameter.
func (r *MemoryAlertRepository) FindOpenByParameter(
	_ context.Context,
	parameterType domain.ParameterType,
) (domain.AlertEvent, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var latest domain.AlertEvent
	found := false

	for _, alert := range r.alerts {
		if alert.ParameterType != parameterType || !isOpenAlert(alert) {
			continue
		}

		if !found || alert.CreatedAt.After(latest.CreatedAt) ||
			alert.CreatedAt.Equal(latest.CreatedAt) && alert.ID > latest.ID {
			latest = alert
			found = true
		}
	}

	return latest, found, nil
}

// UpdateOpen updates current open alert without creating duplicates.
func (r *MemoryAlertRepository) UpdateOpen(
	_ context.Context,
	alert domain.AlertEvent,
) (domain.AlertEvent, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.alerts {
		if r.alerts[i].ID != alert.ID || !isOpenAlert(r.alerts[i]) {
			continue
		}

		r.alerts[i].Level = alert.Level
		r.alerts[i].Value = alert.Value
		r.alerts[i].Unit = alert.Unit
		r.alerts[i].SourceID = alert.SourceID
		r.alerts[i].Message = alert.Message

		return r.alerts[i], true, nil
	}

	return domain.AlertEvent{}, false, nil
}

// ResolveOpenByParameter resolves all open alerts for the given parameter.
func (r *MemoryAlertRepository) ResolveOpenByParameter(
	_ context.Context,
	parameterType domain.ParameterType,
) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	var resolvedCount int64

	for i := range r.alerts {
		if r.alerts[i].ParameterType != parameterType || !isOpenAlert(r.alerts[i]) {
			continue
		}

		r.alerts[i].Resolve(now)
		resolvedCount++
	}

	return resolvedCount, nil
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

func isOpenAlert(alert domain.AlertEvent) bool {
	return alert.Status == domain.AlertStatusActive ||
		alert.Status == domain.AlertStatusAcknowledged
}
