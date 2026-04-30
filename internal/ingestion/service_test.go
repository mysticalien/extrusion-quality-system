package ingestion

import (
	"context"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/storage"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestServiceProcessNormalReading(t *testing.T) {
	ctx := context.Background()
	service, telemetryRepository, alertRepository, qualityRepository := newTestService()

	result, err := service.Process(ctx, TelemetryInput{
		ParameterType: domain.ParameterPressure,
		Value:         65,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("process telemetry: %v", err)
	}

	if result.State != domain.ParameterStateNormal {
		t.Fatalf("expected state %q, got %q", domain.ParameterStateNormal, result.State)
	}

	if result.AlertCreated {
		t.Fatalf("expected alertCreated false")
	}

	readings, err := telemetryRepository.All(ctx)
	if err != nil {
		t.Fatalf("load readings: %v", err)
	}

	if len(readings) != 1 {
		t.Fatalf("expected 1 reading, got %d", len(readings))
	}

	alerts, err := alertRepository.All(ctx)
	if err != nil {
		t.Fatalf("load alerts: %v", err)
	}

	if len(alerts) != 0 {
		t.Fatalf("expected no alerts, got %d", len(alerts))
	}

	qualityIndex, found, err := qualityRepository.Latest(ctx)
	if err != nil {
		t.Fatalf("load latest quality index: %v", err)
	}

	if !found {
		t.Fatalf("expected saved quality index")
	}

	if qualityIndex.Value != 100 {
		t.Fatalf("expected quality index 100, got %.2f", qualityIndex.Value)
	}
}

func TestServiceProcessWarningReading(t *testing.T) {
	ctx := context.Background()
	service, _, alertRepository, qualityRepository := newTestService()

	result, err := service.Process(ctx, TelemetryInput{
		ParameterType: domain.ParameterPressure,
		Value:         82.5,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("process telemetry: %v", err)
	}

	if result.State != domain.ParameterStateWarning {
		t.Fatalf("expected state %q, got %q", domain.ParameterStateWarning, result.State)
	}

	if !result.AlertCreated {
		t.Fatalf("expected alertCreated true")
	}

	if result.AlertLevel == nil || *result.AlertLevel != domain.AlertLevelWarning {
		t.Fatalf("expected warning alert level")
	}

	alerts, err := alertRepository.All(ctx)
	if err != nil {
		t.Fatalf("load alerts: %v", err)
	}

	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}

	qualityIndex, found, err := qualityRepository.Latest(ctx)
	if err != nil {
		t.Fatalf("load latest quality index: %v", err)
	}

	if !found {
		t.Fatalf("expected saved quality index")
	}

	if qualityIndex.Value != 85 {
		t.Fatalf("expected quality index 85, got %.2f", qualityIndex.Value)
	}
}

func TestServiceProcessCriticalReading(t *testing.T) {
	ctx := context.Background()
	service, _, alertRepository, qualityRepository := newTestService()

	result, err := service.Process(ctx, TelemetryInput{
		ParameterType: domain.ParameterPressure,
		Value:         95,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("process telemetry: %v", err)
	}

	if result.State != domain.ParameterStateCritical {
		t.Fatalf("expected state %q, got %q", domain.ParameterStateCritical, result.State)
	}

	if !result.AlertCreated {
		t.Fatalf("expected alertCreated true")
	}

	if result.AlertLevel == nil || *result.AlertLevel != domain.AlertLevelCritical {
		t.Fatalf("expected critical alert level")
	}

	alerts, err := alertRepository.All(ctx)
	if err != nil {
		t.Fatalf("load alerts: %v", err)
	}

	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}

	qualityIndex, found, err := qualityRepository.Latest(ctx)
	if err != nil {
		t.Fatalf("load latest quality index: %v", err)
	}

	if !found {
		t.Fatalf("expected saved quality index")
	}

	if qualityIndex.Value != 70 {
		t.Fatalf("expected quality index 70, got %.2f", qualityIndex.Value)
	}
}

func TestServiceProcessInvalidInput(t *testing.T) {
	ctx := context.Background()
	service, telemetryRepository, alertRepository, qualityRepository := newTestService()

	_, err := service.Process(ctx, TelemetryInput{
		ParameterType: domain.ParameterPressure,
		Value:         65,
		Unit:          domain.UnitPercent,
		SourceID:      domain.SourceID("simulator"),
		MeasuredAt:    time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected error")
	}

	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %T", err)
	}

	readings, err := telemetryRepository.All(ctx)
	if err != nil {
		t.Fatalf("load readings: %v", err)
	}

	if len(readings) != 0 {
		t.Fatalf("expected no readings for invalid input")
	}

	alerts, err := alertRepository.All(ctx)
	if err != nil {
		t.Fatalf("load alerts: %v", err)
	}

	if len(alerts) != 0 {
		t.Fatalf("expected no alerts for invalid input")
	}

	_, found, err := qualityRepository.Latest(ctx)
	if err != nil {
		t.Fatalf("load latest quality index: %v", err)
	}

	if found {
		t.Fatalf("expected no quality index for invalid input")
	}
}

func newTestService() (
	*Service,
	*storage.MemoryTelemetryRepository,
	*storage.MemoryAlertRepository,
	*storage.MemoryQualityRepository,
) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	telemetryRepository := storage.NewMemoryTelemetryRepository()
	alertRepository := storage.NewMemoryAlertRepository()
	qualityRepository := storage.NewMemoryQualityRepository()

	setpoints := map[domain.ParameterType]domain.Setpoint{
		domain.ParameterPressure: {
			ParameterType: domain.ParameterPressure,
			Unit:          domain.UnitBar,
			WarningMin:    30,
			NormalMin:     40,
			NormalMax:     75,
			WarningMax:    90,
		},
	}

	service := NewService(
		logger,
		telemetryRepository,
		alertRepository,
		qualityRepository,
		setpoints,
	)

	return service, telemetryRepository, alertRepository, qualityRepository
}
