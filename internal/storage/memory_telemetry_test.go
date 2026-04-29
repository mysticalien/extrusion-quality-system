package storage

import (
	"extrusion-quality-system/internal/domain"
	"testing"
	"time"
)

func TestMemoryTelemetryStoreSaveAndAll(t *testing.T) {
	store := NewMemoryTelemetryStore()

	reading, err := store.Save(domain.TelemetryReading{
		ParameterType: domain.ParameterPressure,
		Value:         65,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("save reading: %v", err)
	}

	if reading.ID != 1 {
		t.Fatalf("expected id 1, got %d", reading.ID)
	}

	readings, err := store.All()
	if err != nil {
		t.Fatalf("load readings: %v", err)
	}

	if len(readings) != 1 {
		t.Fatalf("expected 1 reading, got %d", len(readings))
	}

	if readings[0].ParameterType != domain.ParameterPressure {
		t.Fatalf("expected parameter %q, got %q", domain.ParameterPressure, readings[0].ParameterType)
	}
}
