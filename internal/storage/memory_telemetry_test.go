package storage

import (
	"context"
	"extrusion-quality-system/internal/domain"
	"testing"
	"time"
)

func TestMemoryTelemetryRepositorySaveAndAll(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryTelemetryRepository()

	measuredAt := time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC)

	reading, err := repository.Save(ctx, domain.TelemetryReading{
		ParameterType: domain.ParameterPressure,
		Value:         65,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    measuredAt,
		CreatedAt:     measuredAt,
	})
	if err != nil {
		t.Fatalf("save reading: %v", err)
	}

	if reading.ID != 1 {
		t.Fatalf("expected id 1, got %d", reading.ID)
	}

	readings, err := repository.All(ctx)
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

func TestMemoryTelemetryRepositoryLatest(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryTelemetryRepository()

	firstMeasuredAt := time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC)
	secondMeasuredAt := time.Date(2026, 4, 27, 18, 5, 0, 0, time.UTC)

	_, err := repository.Save(ctx, domain.TelemetryReading{
		ParameterType: domain.ParameterPressure,
		Value:         60,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    firstMeasuredAt,
		CreatedAt:     firstMeasuredAt,
	})
	if err != nil {
		t.Fatalf("save first pressure reading: %v", err)
	}

	latestPressureReading, err := repository.Save(ctx, domain.TelemetryReading{
		ParameterType: domain.ParameterPressure,
		Value:         82.5,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    secondMeasuredAt,
		CreatedAt:     secondMeasuredAt,
	})
	if err != nil {
		t.Fatalf("save latest pressure reading: %v", err)
	}

	latestMoistureReading, err := repository.Save(ctx, domain.TelemetryReading{
		ParameterType: domain.ParameterMoisture,
		Value:         25,
		Unit:          domain.UnitPercent,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    firstMeasuredAt,
		CreatedAt:     firstMeasuredAt,
	})
	if err != nil {
		t.Fatalf("save moisture reading: %v", err)
	}

	latest, err := repository.Latest(ctx)
	if err != nil {
		t.Fatalf("load latest readings: %v", err)
	}

	if len(latest) != 2 {
		t.Fatalf("expected 2 latest readings, got %d", len(latest))
	}

	foundPressure := false
	foundMoisture := false

	for _, reading := range latest {
		switch reading.ParameterType {
		case domain.ParameterPressure:
			foundPressure = true
			if reading.ID != latestPressureReading.ID {
				t.Fatalf("expected latest pressure id %d, got %d", latestPressureReading.ID, reading.ID)
			}
		case domain.ParameterMoisture:
			foundMoisture = true
			if reading.ID != latestMoistureReading.ID {
				t.Fatalf("expected latest moisture id %d, got %d", latestMoistureReading.ID, reading.ID)
			}
		}
	}

	if !foundPressure {
		t.Fatalf("expected latest pressure reading")
	}

	if !foundMoisture {
		t.Fatalf("expected latest moisture reading")
	}
}

func TestMemoryTelemetryRepositoryHistoryByParameter(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryTelemetryRepository()

	from := time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC)
	middle := time.Date(2026, 4, 27, 18, 5, 0, 0, time.UTC)
	to := time.Date(2026, 4, 27, 18, 10, 0, 0, time.UTC)

	_, err := repository.Save(ctx, domain.TelemetryReading{
		ParameterType: domain.ParameterPressure,
		Value:         60,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    from,
		CreatedAt:     from,
	})
	if err != nil {
		t.Fatalf("save first pressure reading: %v", err)
	}

	expectedReading, err := repository.Save(ctx, domain.TelemetryReading{
		ParameterType: domain.ParameterPressure,
		Value:         82.5,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    middle,
		CreatedAt:     middle,
	})
	if err != nil {
		t.Fatalf("save second pressure reading: %v", err)
	}

	_, err = repository.Save(ctx, domain.TelemetryReading{
		ParameterType: domain.ParameterMoisture,
		Value:         25,
		Unit:          domain.UnitPercent,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    middle,
		CreatedAt:     middle,
	})
	if err != nil {
		t.Fatalf("save moisture reading: %v", err)
	}

	history, err := repository.HistoryByParameter(
		ctx,
		domain.ParameterPressure,
		middle,
		to,
		10,
	)
	if err != nil {
		t.Fatalf("load pressure history: %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("expected 1 history reading, got %d", len(history))
	}

	if history[0].ID != expectedReading.ID {
		t.Fatalf("expected reading id %d, got %d", expectedReading.ID, history[0].ID)
	}
}
