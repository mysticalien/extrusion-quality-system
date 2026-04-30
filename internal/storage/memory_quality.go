package storage

import (
	"context"
	"sync"
	"time"

	"extrusion-quality-system/internal/domain"
)

// MemoryQualityRepository stores quality index values in memory.
// It is used for tests and early prototypes.
type MemoryQualityRepository struct {
	mu      sync.RWMutex
	nextID  domain.QualityIndexID
	indexes []domain.QualityIndex
}

// NewMemoryQualityRepository creates an empty in-memory quality repository.
func NewMemoryQualityRepository() *MemoryQualityRepository {
	return &MemoryQualityRepository{
		nextID: 1,
	}
}

// Save stores a quality index value and assigns an in-memory ID.
func (r *MemoryQualityRepository) Save(
	_ context.Context,
	index domain.QualityIndex,
) (domain.QualityIndex, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	index.ID = r.nextID
	r.nextID++

	r.indexes = append(r.indexes, index)

	return index, nil
}

// Latest returns the latest stored quality index value.
func (r *MemoryQualityRepository) Latest(_ context.Context) (domain.QualityIndex, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.indexes) == 0 {
		return domain.QualityIndex{}, false, nil
	}

	return r.indexes[len(r.indexes)-1], true, nil
}

// History returns quality index values in the given time range.
func (r *MemoryQualityRepository) History(
	_ context.Context,
	from time.Time,
	to time.Time,
	limit int,
) ([]domain.QualityIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	result := make([]domain.QualityIndex, 0)

	for _, index := range r.indexes {
		if !from.IsZero() && index.CalculatedAt.Before(from) {
			continue
		}

		if !to.IsZero() && index.CalculatedAt.After(to) {
			continue
		}

		result = append(result, index)

		if len(result) >= limit {
			break
		}
	}

	return result, nil
}
