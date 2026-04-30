package mqttingest

import (
	"context"
	"encoding/json"
	"extrusion-quality-system/internal/config"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/ingestion"
	"io"
	"log/slog"
	"testing"
	"time"
)

type fakeTelemetrySink struct {
	inputs []ingestion.TelemetryInput
}

func (s *fakeTelemetrySink) Submit(_ context.Context, input ingestion.TelemetryInput) error {
	s.inputs = append(s.inputs, input)
	return nil
}

func TestSubscriberHandlePayloadSubmitsTelemetry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	sink := &fakeTelemetrySink{}

	subscriber := NewSubscriber(logger, config.MQTTConfig{}, sink)

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

	if err := subscriber.handlePayload(context.Background(), payload); err != nil {
		t.Fatalf("handle payload: %v", err)
	}

	if len(sink.inputs) != 1 {
		t.Fatalf("expected 1 submitted input, got %d", len(sink.inputs))
	}

	if sink.inputs[0].ParameterType != domain.ParameterPressure {
		t.Fatalf("expected parameter %q, got %q", domain.ParameterPressure, sink.inputs[0].ParameterType)
	}
}

func TestSubscriberHandleInvalidPayload(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	sink := &fakeTelemetrySink{}

	subscriber := NewSubscriber(logger, config.MQTTConfig{}, sink)

	err := subscriber.handlePayload(context.Background(), []byte(`{`))
	if err == nil {
		t.Fatalf("expected error")
	}
}
