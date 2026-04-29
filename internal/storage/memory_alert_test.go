package storage

import (
	"extrusion-quality-system/internal/domain"
	"testing"
	"time"
)

func TestMemoryAlertStoreCreateAllAndActive(t *testing.T) {
	store := NewMemoryAlertStore()

	activeAlert, err := store.Create(domain.AlertEvent{
		ParameterType: domain.ParameterPressure,
		Level:         domain.AlertLevelWarning,
		Status:        domain.AlertStatusActive,
		Value:         82.5,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		Message:       "pressure warning",
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create active alert: %v", err)
	}

	resolvedAlert, err := store.Create(domain.AlertEvent{
		ParameterType: domain.ParameterPressure,
		Level:         domain.AlertLevelCritical,
		Status:        domain.AlertStatusResolved,
		Value:         95,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		Message:       "pressure critical",
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create resolved alert: %v", err)
	}

	if activeAlert.ID != 1 {
		t.Fatalf("expected active alert id 1, got %d", activeAlert.ID)
	}

	if resolvedAlert.ID != 2 {
		t.Fatalf("expected resolved alert id 2, got %d", resolvedAlert.ID)
	}

	all, err := store.All()
	if err != nil {
		t.Fatalf("load all alerts: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(all))
	}

	active, err := store.Active()
	if err != nil {
		t.Fatalf("load active alerts: %v", err)
	}

	if len(active) != 1 {
		t.Fatalf("expected 1 active alert, got %d", len(active))
	}

	if active[0].ID != activeAlert.ID {
		t.Fatalf("expected active alert id %d, got %d", activeAlert.ID, active[0].ID)
	}
}

func TestMemoryAlertStoreAcknowledge(t *testing.T) {
	store := NewMemoryAlertStore()

	alert, err := store.Create(domain.AlertEvent{
		ParameterType: domain.ParameterPressure,
		Level:         domain.AlertLevelWarning,
		Status:        domain.AlertStatusActive,
		Value:         82.5,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		Message:       "pressure warning",
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create alert: %v", err)
	}

	updated, found, err := store.Acknowledge(alert.ID, nil)
	if err != nil {
		t.Fatalf("acknowledge alert: %v", err)
	}

	if !found {
		t.Fatalf("expected alert to be found")
	}

	if updated.Status != domain.AlertStatusAcknowledged {
		t.Fatalf("expected status %q, got %q", domain.AlertStatusAcknowledged, updated.Status)
	}

	if updated.AcknowledgedAt == nil {
		t.Fatalf("expected acknowledgedAt to be set")
	}
}

func TestMemoryAlertStoreResolve(t *testing.T) {
	store := NewMemoryAlertStore()

	alert, err := store.Create(domain.AlertEvent{
		ParameterType: domain.ParameterPressure,
		Level:         domain.AlertLevelCritical,
		Status:        domain.AlertStatusActive,
		Value:         95,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		Message:       "pressure critical",
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create alert: %v", err)
	}

	updated, found, err := store.Resolve(alert.ID)
	if err != nil {
		t.Fatalf("resolve alert: %v", err)
	}

	if !found {
		t.Fatalf("expected alert to be found")
	}

	if updated.Status != domain.AlertStatusResolved {
		t.Fatalf("expected status %q, got %q", domain.AlertStatusResolved, updated.Status)
	}

	if updated.ResolvedAt == nil {
		t.Fatalf("expected resolvedAt to be set")
	}
}

func TestMemoryAlertStoreNotFound(t *testing.T) {
	store := NewMemoryAlertStore()

	_, found, err := store.Acknowledge(999, nil)
	if err != nil {
		t.Fatalf("acknowledge alert: %v", err)
	}

	if found {
		t.Fatalf("expected alert not to be found")
	}

	_, found, err = store.Resolve(999)
	if err != nil {
		t.Fatalf("resolve alert: %v", err)
	}

	if found {
		t.Fatalf("expected alert not to be found")
	}
}
