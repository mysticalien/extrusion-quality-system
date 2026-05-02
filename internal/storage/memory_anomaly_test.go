package storage

import (
	"context"
	"testing"
	"time"

	"extrusion-quality-system/internal/domain"
)

func TestMemoryAnomalyRepositoryCreateAndActive(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryAnomalyRepository()

	created, err := repository.Create(ctx, domain.AnomalyEvent{
		Type:          domain.AnomalyTypeJump,
		ParameterType: domain.ParameterPressure,
		Level:         domain.AlertLevelWarning,
		Status:        domain.AlertStatusActive,
		Message:       "pressure jump",
		SourceID:      domain.SourceID("test-simulator"),
		ObservedAt:    time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create anomaly: %v", err)
	}

	if created.ID == 0 {
		t.Fatalf("expected anomaly id to be set")
	}

	active, err := repository.Active(ctx)
	if err != nil {
		t.Fatalf("load active anomalies: %v", err)
	}

	if len(active) != 1 {
		t.Fatalf("expected 1 active anomaly, got %d", len(active))
	}

	if active[0].Type != domain.AnomalyTypeJump {
		t.Fatalf("expected anomaly type %q, got %q", domain.AnomalyTypeJump, active[0].Type)
	}
}

func TestMemoryAnomalyRepositoryUpdateOpenDoesNotCreateDuplicate(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryAnomalyRepository()

	created, err := repository.Create(ctx, domain.AnomalyEvent{
		Type:          domain.AnomalyTypeDrift,
		ParameterType: domain.ParameterMoisture,
		Level:         domain.AlertLevelWarning,
		Status:        domain.AlertStatusActive,
		Message:       "moisture drift",
		SourceID:      domain.SourceID("test-simulator"),
		ObservedAt:    time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create anomaly: %v", err)
	}

	currentValue := 24.5

	created.CurrentValue = &currentValue
	created.Message = "updated moisture drift"

	updated, ok, err := repository.UpdateOpen(ctx, created)
	if err != nil {
		t.Fatalf("update open anomaly: %v", err)
	}

	if !ok {
		t.Fatalf("expected anomaly to be updated")
	}

	if updated.Message != "updated moisture drift" {
		t.Fatalf("expected updated message, got %q", updated.Message)
	}

	active, err := repository.Active(ctx)
	if err != nil {
		t.Fatalf("load active anomalies: %v", err)
	}

	if len(active) != 1 {
		t.Fatalf("expected 1 active anomaly after update, got %d", len(active))
	}
}

func TestMemoryAnomalyRepositoryResolveOpenByTypeAndParameter(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryAnomalyRepository()

	_, err := repository.Create(ctx, domain.AnomalyEvent{
		Type:          domain.AnomalyTypeCombinedRisk,
		ParameterType: domain.ParameterProcessRisk,
		Level:         domain.AlertLevelCritical,
		Status:        domain.AlertStatusActive,
		Message:       "combined risk",
		SourceID:      domain.SourceID("test-simulator"),
		ObservedAt:    time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create anomaly: %v", err)
	}

	resolvedCount, err := repository.ResolveOpenByTypeAndParameter(
		ctx,
		domain.AnomalyTypeCombinedRisk,
		domain.ParameterProcessRisk,
	)
	if err != nil {
		t.Fatalf("resolve open anomaly: %v", err)
	}

	if resolvedCount != 1 {
		t.Fatalf("expected 1 resolved anomaly, got %d", resolvedCount)
	}

	active, err := repository.Active(ctx)
	if err != nil {
		t.Fatalf("load active anomalies: %v", err)
	}

	if len(active) != 0 {
		t.Fatalf("expected no active anomalies, got %d", len(active))
	}
}
