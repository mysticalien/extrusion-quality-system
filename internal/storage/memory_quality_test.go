package storage

import (
	"extrusion-quality-system/internal/domain"
	"testing"
	"time"
)

func TestMemoryQualityStoreLatestEmpty(t *testing.T) {
	store := NewMemoryQualityStore()

	_, found, err := store.Latest()
	if err != nil {
		t.Fatalf("load latest quality index: %v", err)
	}

	if found {
		t.Fatalf("expected latest quality index not to be found")
	}
}

func TestMemoryQualityStoreSaveAndLatest(t *testing.T) {
	store := NewMemoryQualityStore()

	first, err := store.Save(domain.QualityIndex{
		Value:            85,
		State:            domain.QualityStateStable,
		ParameterPenalty: 15,
		AnomalyPenalty:   0,
		CalculatedAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("save first quality index: %v", err)
	}

	second, err := store.Save(domain.QualityIndex{
		Value:            55,
		State:            domain.QualityStateUnstable,
		ParameterPenalty: 45,
		AnomalyPenalty:   0,
		CalculatedAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("save second quality index: %v", err)
	}

	if first.ID != 1 {
		t.Fatalf("expected first id 1, got %d", first.ID)
	}

	if second.ID != 2 {
		t.Fatalf("expected second id 2, got %d", second.ID)
	}

	latest, found, err := store.Latest()
	if err != nil {
		t.Fatalf("load latest quality index: %v", err)
	}

	if !found {
		t.Fatalf("expected latest quality index to be found")
	}

	if latest.ID != second.ID {
		t.Fatalf("expected latest id %d, got %d", second.ID, latest.ID)
	}

	if latest.Value != second.Value {
		t.Fatalf("expected latest value %.2f, got %.2f", second.Value, latest.Value)
	}
}
