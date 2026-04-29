package storage

import (
	"sync"

	"extrusion-quality-system/internal/domain"
)

// MemoryTelemetryStore stores telemetry readings in memory.
// It is used only for the first prototype and tests.
type MemoryTelemetryStore struct {
	mu       sync.RWMutex
	nextID   domain.TelemetryReadingID
	readings []domain.TelemetryReading
}

// NewMemoryTelemetryStore creates an empty in-memory telemetry store.
func NewMemoryTelemetryStore() *MemoryTelemetryStore {
	return &MemoryTelemetryStore{
		nextID: 1,
	}
}

// Save stores a telemetry reading and assigns an in-memory ID.
func (s *MemoryTelemetryStore) Save(reading domain.TelemetryReading) (domain.TelemetryReading, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reading.ID = s.nextID
	s.nextID++

	s.readings = append(s.readings, reading)

	return reading, nil
}

// All returns a copy of all stored telemetry readings.
func (s *MemoryTelemetryStore) All() ([]domain.TelemetryReading, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]domain.TelemetryReading, len(s.readings))
	copy(result, s.readings)

	return result, nil
}
