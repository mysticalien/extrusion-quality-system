package storage

import (
	"sync"

	"extrusion-quality-system/internal/domain"
)

// MemoryQualityStore stores quality index values in memory.
// It is used only for tests and early prototypes.
type MemoryQualityStore struct {
	mu      sync.RWMutex
	nextID  domain.QualityIndexID
	indexes []domain.QualityIndex
}

// NewMemoryQualityStore creates an empty in-memory quality index store.
func NewMemoryQualityStore() *MemoryQualityStore {
	return &MemoryQualityStore{
		nextID: 1,
	}
}

// Save stores a quality index value and assigns an in-memory ID.
func (s *MemoryQualityStore) Save(index domain.QualityIndex) (domain.QualityIndex, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index.ID = s.nextID
	s.nextID++

	s.indexes = append(s.indexes, index)

	return index, nil
}

// Latest returns the latest stored quality index value.
func (s *MemoryQualityStore) Latest() (domain.QualityIndex, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.indexes) == 0 {
		return domain.QualityIndex{}, false, nil
	}

	return s.indexes[len(s.indexes)-1], true, nil
}
