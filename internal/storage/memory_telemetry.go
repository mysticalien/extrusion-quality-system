package storage

import (
	"context"
	"sort"
	"sync"
	"time"

	"extrusion-quality-system/internal/domain"
)

// MemoryTelemetryRepository stores telemetry readings in memory.
// It is used for tests and early prototypes.
type MemoryTelemetryRepository struct {
	mu       sync.RWMutex
	nextID   domain.TelemetryReadingID
	readings []domain.TelemetryReading
}

// NewMemoryTelemetryRepository creates an empty in-memory telemetry repository.
func NewMemoryTelemetryRepository() *MemoryTelemetryRepository {
	return &MemoryTelemetryRepository{
		nextID: 1,
	}
}

// Save stores a telemetry reading and assigns an in-memory ID.
func (r *MemoryTelemetryRepository) Save(
	_ context.Context,
	reading domain.TelemetryReading,
) (domain.TelemetryReading, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	reading.ID = r.nextID
	r.nextID++

	r.readings = append(r.readings, reading)

	return reading, nil
}

// All returns a copy of all stored telemetry readings.
func (r *MemoryTelemetryRepository) All(_ context.Context) ([]domain.TelemetryReading, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.TelemetryReading, len(r.readings))
	copy(result, r.readings)

	return result, nil
}

// Latest returns the latest reading for each parameter type.
func (r *MemoryTelemetryRepository) Latest(_ context.Context) ([]domain.TelemetryReading, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	latestByParameter := make(map[domain.ParameterType]domain.TelemetryReading)

	for _, reading := range r.readings {
		current, exists := latestByParameter[reading.ParameterType]
		if !exists || reading.MeasuredAt.After(current.MeasuredAt) ||
			reading.MeasuredAt.Equal(current.MeasuredAt) && reading.ID > current.ID {
			latestByParameter[reading.ParameterType] = reading
		}
	}

	result := make([]domain.TelemetryReading, 0, len(latestByParameter))
	for _, reading := range latestByParameter {
		result = append(result, reading)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ParameterType < result[j].ParameterType
	})

	return result, nil
}

// HistoryByParameter returns telemetry readings for one parameter in the given time range.
func (r *MemoryTelemetryRepository) HistoryByParameter(
	_ context.Context,
	parameterType domain.ParameterType,
	from time.Time,
	to time.Time,
	limit int,
) ([]domain.TelemetryReading, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	result := make([]domain.TelemetryReading, 0)

	for _, reading := range r.readings {
		if reading.ParameterType != parameterType {
			continue
		}

		if !from.IsZero() && reading.MeasuredAt.Before(from) {
			continue
		}

		if !to.IsZero() && reading.MeasuredAt.After(to) {
			continue
		}

		result = append(result, reading)

		if len(result) >= limit {
			break
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].MeasuredAt.Equal(result[j].MeasuredAt) {
			return result[i].ID < result[j].ID
		}

		return result[i].MeasuredAt.Before(result[j].MeasuredAt)
	})

	return result, nil
}
