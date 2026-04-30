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

func TestAsyncProcessorProcessesTelemetryInBackground(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	processor := NewAsyncProcessor(logger, service, 2, 10)
	processor.Start(ctx)

	err := processor.Submit(ctx, TelemetryInput{
		ParameterType: domain.ParameterPressure,
		Value:         82.5,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("mqtt-simulator"),
		MeasuredAt:    time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("submit telemetry: %v", err)
	}

	waitUntil(t, time.Second, func() bool {
		alerts, err := alertRepository.Active(context.Background())
		if err != nil {
			return false
		}

		return len(alerts) == 1
	})

	alerts, err := alertRepository.Active(context.Background())
	if err != nil {
		t.Fatalf("load active alerts: %v", err)
	}

	if alerts[0].Level != domain.AlertLevelWarning {
		t.Fatalf("expected warning alert, got %q", alerts[0].Level)
	}
}

func waitUntil(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("condition was not met within %s", timeout)
}
