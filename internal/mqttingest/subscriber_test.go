package mqttingest

import (
	"context"
	"encoding/json"
	"extrusion-quality-system/internal/config"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/ingestion"
	"extrusion-quality-system/internal/storage"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestSubscriberHandlePayload(t *testing.T) {
	ctx := context.Background()

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

	service := ingestion.NewService(
		logger,
		telemetryRepository,
		alertRepository,
		qualityRepository,
		setpoints,
	)

	subscriber := NewSubscriber(logger, config.MQTTConfig{}, service)

	payload, err := json.Marshal(ingestion.TelemetryInput{
		ParameterType: domain.ParameterPressure,
		Value:         82.5,
		Unit:          domain.UnitBar,
		SourceID:      domain.SourceID("mqtt-simulator"),
		MeasuredAt:    time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	if err := subscriber.handlePayload(ctx, payload); err != nil {
		t.Fatalf("handle payload: %v", err)
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

	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}

	if alerts[0].Level != domain.AlertLevelWarning {
		t.Fatalf("expected warning alert, got %q", alerts[0].Level)
	}
}

func TestSubscriberHandleInvalidPayload(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := ingestion.NewService(
		logger,
		storage.NewMemoryTelemetryRepository(),
		storage.NewMemoryAlertRepository(),
		storage.NewMemoryQualityRepository(),
		nil,
	)

	subscriber := NewSubscriber(logger, config.MQTTConfig{}, service)

	err := subscriber.handlePayload(context.Background(), []byte(`{`))
	if err == nil {
		t.Fatalf("expected error")
	}
}
