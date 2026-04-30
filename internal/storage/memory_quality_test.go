package storage

import (
	"context"
	"extrusion-quality-system/internal/domain"
	"testing"
	"time"
)

func TestMemoryQualityRepositoryLatestEmpty(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryQualityRepository()

	_, found, err := repository.Latest(ctx)
	if err != nil {
		t.Fatalf("load latest quality index: %v", err)
	}

	if found {
		t.Fatalf("expected latest quality index not to be found")
	}
}

func TestMemoryQualityRepositorySaveAndLatest(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryQualityRepository()

	first, err := repository.Save(ctx, domain.QualityIndex{
		Value:            85,
		State:            domain.QualityStateStable,
		ParameterPenalty: 15,
		AnomalyPenalty:   0,
		CalculatedAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("save first quality index: %v", err)
	}

	second, err := repository.Save(ctx, domain.QualityIndex{
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

	latest, found, err := repository.Latest(ctx)
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

func TestMemoryQualityRepositoryHistory(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryQualityRepository()

	firstTime := time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC)
	secondTime := time.Date(2026, 4, 27, 18, 5, 0, 0, time.UTC)
	thirdTime := time.Date(2026, 4, 27, 18, 10, 0, 0, time.UTC)

	_, err := repository.Save(ctx, domain.QualityIndex{
		Value:            100,
		State:            domain.QualityStateStable,
		ParameterPenalty: 0,
		AnomalyPenalty:   0,
		CalculatedAt:     firstTime,
	})
	if err != nil {
		t.Fatalf("save first quality index: %v", err)
	}

	expectedIndex, err := repository.Save(ctx, domain.QualityIndex{
		Value:            85,
		State:            domain.QualityStateStable,
		ParameterPenalty: 15,
		AnomalyPenalty:   0,
		CalculatedAt:     secondTime,
	})
	if err != nil {
		t.Fatalf("save second quality index: %v", err)
	}

	_, err = repository.Save(ctx, domain.QualityIndex{
		Value:            55,
		State:            domain.QualityStateUnstable,
		ParameterPenalty: 45,
		AnomalyPenalty:   0,
		CalculatedAt:     thirdTime,
	})
	if err != nil {
		t.Fatalf("save third quality index: %v", err)
	}

	history, err := repository.History(ctx, secondTime, secondTime, 10)
	if err != nil {
		t.Fatalf("load quality history: %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("expected 1 quality index, got %d", len(history))
	}

	if history[0].ID != expectedIndex.ID {
		t.Fatalf("expected quality index id %d, got %d", expectedIndex.ID, history[0].ID)
	}
}
