package storage

import (
	"context"
	"sort"
	"sync"
	"time"

	"extrusion-quality-system/internal/domain"
)

type MemoryAnomalyRepository struct {
	mu        sync.RWMutex
	nextID    domain.AnomalyID
	anomalies []domain.AnomalyEvent
}

func NewMemoryAnomalyRepository() *MemoryAnomalyRepository {
	return &MemoryAnomalyRepository{
		nextID: 1,
	}
}

func (r *MemoryAnomalyRepository) Create(
	_ context.Context,
	anomaly domain.AnomalyEvent,
) (domain.AnomalyEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()

	anomaly.ID = r.nextID
	r.nextID++

	if anomaly.Status == "" {
		anomaly.Status = domain.AlertStatusActive
	}

	if anomaly.CreatedAt.IsZero() {
		anomaly.CreatedAt = now
	}

	if anomaly.UpdatedAt.IsZero() {
		anomaly.UpdatedAt = now
	}

	if anomaly.ObservedAt.IsZero() {
		anomaly.ObservedAt = now
	}

	r.anomalies = append(r.anomalies, anomaly)

	return anomaly, nil
}

func (r *MemoryAnomalyRepository) All(_ context.Context) ([]domain.AnomalyEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.AnomalyEvent, len(r.anomalies))
	copy(result, r.anomalies)

	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID > result[j].ID
		}

		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result, nil
}

func (r *MemoryAnomalyRepository) Active(_ context.Context) ([]domain.AnomalyEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	latestByKey := make(map[string]domain.AnomalyEvent)

	for _, anomaly := range r.anomalies {
		if !isOpenAnomaly(anomaly) {
			continue
		}

		key := anomalyRepositoryKey(anomaly.Type, anomaly.ParameterType)

		current, exists := latestByKey[key]
		if !exists ||
			anomaly.UpdatedAt.After(current.UpdatedAt) ||
			anomaly.UpdatedAt.Equal(current.UpdatedAt) && anomaly.ID > current.ID {
			latestByKey[key] = anomaly
		}
	}

	result := make([]domain.AnomalyEvent, 0, len(latestByKey))
	for _, anomaly := range latestByKey {
		result = append(result, anomaly)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].UpdatedAt.Equal(result[j].UpdatedAt) {
			return result[i].ID > result[j].ID
		}

		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})

	return result, nil
}

func (r *MemoryAnomalyRepository) FindOpenByTypeAndParameter(
	_ context.Context,
	anomalyType domain.AnomalyType,
	parameterType domain.ParameterType,
) (domain.AnomalyEvent, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var latest domain.AnomalyEvent
	found := false

	for _, anomaly := range r.anomalies {
		if anomaly.Type != anomalyType ||
			anomaly.ParameterType != parameterType ||
			!isOpenAnomaly(anomaly) {
			continue
		}

		if !found ||
			anomaly.UpdatedAt.After(latest.UpdatedAt) ||
			anomaly.UpdatedAt.Equal(latest.UpdatedAt) && anomaly.ID > latest.ID {
			latest = anomaly
			found = true
		}
	}

	return latest, found, nil
}

func (r *MemoryAnomalyRepository) UpdateOpen(
	_ context.Context,
	anomaly domain.AnomalyEvent,
) (domain.AnomalyEvent, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()

	for i := range r.anomalies {
		if r.anomalies[i].ID != anomaly.ID || !isOpenAnomaly(r.anomalies[i]) {
			continue
		}

		r.anomalies[i].Level = anomaly.Level
		r.anomalies[i].Message = anomaly.Message
		r.anomalies[i].CurrentValue = anomaly.CurrentValue
		r.anomalies[i].PreviousValue = anomaly.PreviousValue
		r.anomalies[i].SourceID = anomaly.SourceID
		r.anomalies[i].ObservedAt = anomaly.ObservedAt
		r.anomalies[i].UpdatedAt = now

		return r.anomalies[i], true, nil
	}

	return domain.AnomalyEvent{}, false, nil
}

func (r *MemoryAnomalyRepository) ResolveOpenByTypeAndParameter(
	_ context.Context,
	anomalyType domain.AnomalyType,
	parameterType domain.ParameterType,
) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	var resolvedCount int64

	for i := range r.anomalies {
		if r.anomalies[i].Type != anomalyType ||
			r.anomalies[i].ParameterType != parameterType ||
			!isOpenAnomaly(r.anomalies[i]) {
			continue
		}

		r.anomalies[i].Status = domain.AlertStatusResolved
		r.anomalies[i].ResolvedAt = &now
		r.anomalies[i].UpdatedAt = now

		resolvedCount++
	}

	return resolvedCount, nil
}

func isOpenAnomaly(anomaly domain.AnomalyEvent) bool {
	return anomaly.Status == domain.AlertStatusActive ||
		anomaly.Status == domain.AlertStatusAcknowledged
}

func anomalyRepositoryKey(
	anomalyType domain.AnomalyType,
	parameterType domain.ParameterType,
) string {
	return string(anomalyType) + ":" + string(parameterType)
}
