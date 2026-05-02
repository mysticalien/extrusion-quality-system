package anomalies

import (
	"context"
	"testing"
	"time"

	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/storage"
)

func TestDetectorDetectsPressureJumpOnlyWhenPressureRises(t *testing.T) {
	ctx := context.Background()
	telemetryRepository := storage.NewMemoryTelemetryRepository()
	detector := NewDetector(telemetryRepository)

	first := saveReading(t, ctx, telemetryRepository, domain.TelemetryReading{
		ParameterType: domain.ParameterPressure,
		Value:         65,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("test-simulator"),
		MeasuredAt:    time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC),
	})

	second := saveReading(t, ctx, telemetryRepository, domain.TelemetryReading{
		ParameterType: domain.ParameterPressure,
		Value:         82,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("test-simulator"),
		MeasuredAt:    first.MeasuredAt.Add(time.Second),
	})

	anomalies, err := detector.Detect(ctx, second)
	if err != nil {
		t.Fatalf("detect anomalies: %v", err)
	}

	if !hasAnomaly(anomalies, domain.AnomalyTypeJump, domain.ParameterPressure) {
		t.Fatalf("expected pressure jump anomaly, got %#v", anomalies)
	}
}

func TestDetectorDoesNotDetectPressureJumpWhenPressureFalls(t *testing.T) {
	ctx := context.Background()
	telemetryRepository := storage.NewMemoryTelemetryRepository()
	detector := NewDetector(telemetryRepository)

	first := saveReading(t, ctx, telemetryRepository, domain.TelemetryReading{
		ParameterType: domain.ParameterPressure,
		Value:         82.5,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("test-simulator"),
		MeasuredAt:    time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC),
	})

	second := saveReading(t, ctx, telemetryRepository, domain.TelemetryReading{
		ParameterType: domain.ParameterPressure,
		Value:         65,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("test-simulator"),
		MeasuredAt:    first.MeasuredAt.Add(time.Second),
	})

	anomalies, err := detector.Detect(ctx, second)
	if err != nil {
		t.Fatalf("detect anomalies: %v", err)
	}

	if hasAnomaly(anomalies, domain.AnomalyTypeJump, domain.ParameterPressure) {
		t.Fatalf("did not expect pressure jump anomaly when pressure falls, got %#v", anomalies)
	}
}

func TestDetectorDetectsMoistureDriftWhenMoistureFalls(t *testing.T) {
	ctx := context.Background()
	telemetryRepository := storage.NewMemoryTelemetryRepository()
	detector := NewDetector(telemetryRepository)

	baseTime := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)

	values := []float64{26.5, 26.0, 25.5, 25.0, 24.5}

	var current domain.TelemetryReading

	for index, value := range values {
		current = saveReading(t, ctx, telemetryRepository, domain.TelemetryReading{
			ParameterType: domain.ParameterMoisture,
			Value:         value,
			Unit:          domain.UnitPercent,
			SourceID:      domain.SourceID("test-simulator"),
			MeasuredAt:    baseTime.Add(time.Duration(index) * time.Second),
		})
	}

	anomalies, err := detector.Detect(ctx, current)
	if err != nil {
		t.Fatalf("detect anomalies: %v", err)
	}

	if !hasAnomaly(anomalies, domain.AnomalyTypeDrift, domain.ParameterMoisture) {
		t.Fatalf("expected moisture drift anomaly, got %#v", anomalies)
	}
}

func TestDetectorDetectsCombinedRisk(t *testing.T) {
	ctx := context.Background()
	telemetryRepository := storage.NewMemoryTelemetryRepository()
	detector := NewDetector(telemetryRepository)

	baseTime := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)

	pressureValues := []float64{65, 67, 69, 71, 73}
	moistureValues := []float64{26.5, 26.0, 25.5, 25.0, 24.5}
	driveLoadValues := []float64{55, 57, 60, 62, 64}

	var current domain.TelemetryReading

	for index := range pressureValues {
		measuredAt := baseTime.Add(time.Duration(index) * time.Second)

		saveReading(t, ctx, telemetryRepository, domain.TelemetryReading{
			ParameterType: domain.ParameterPressure,
			Value:         pressureValues[index],
			Unit:          domain.UnitBar,
			SourceID:      domain.SourceID("test-simulator"),
			MeasuredAt:    measuredAt,
		})

		saveReading(t, ctx, telemetryRepository, domain.TelemetryReading{
			ParameterType: domain.ParameterMoisture,
			Value:         moistureValues[index],
			Unit:          domain.UnitPercent,
			SourceID:      domain.SourceID("test-simulator"),
			MeasuredAt:    measuredAt,
		})

		current = saveReading(t, ctx, telemetryRepository, domain.TelemetryReading{
			ParameterType: domain.ParameterDriveLoad,
			Value:         driveLoadValues[index],
			Unit:          domain.UnitPercent,
			SourceID:      domain.SourceID("test-simulator"),
			MeasuredAt:    measuredAt,
		})
	}

	anomalies, err := detector.Detect(ctx, current)
	if err != nil {
		t.Fatalf("detect anomalies: %v", err)
	}

	if !hasAnomaly(anomalies, domain.AnomalyTypeCombinedRisk, domain.ParameterProcessRisk) {
		t.Fatalf("expected combined risk anomaly, got %#v", anomalies)
	}
}

func saveReading(
	t *testing.T,
	ctx context.Context,
	repository storage.TelemetryRepository,
	reading domain.TelemetryReading,
) domain.TelemetryReading {
	t.Helper()

	saved, err := repository.Save(ctx, reading)
	if err != nil {
		t.Fatalf("save telemetry reading: %v", err)
	}

	return saved
}

func hasAnomaly(
	anomalies []domain.AnomalyEvent,
	anomalyType domain.AnomalyType,
	parameterType domain.ParameterType,
) bool {
	for _, anomaly := range anomalies {
		if anomaly.Type == anomalyType && anomaly.ParameterType == parameterType {
			return true
		}
	}

	return false
}
