package main

import (
	"extrusion-quality-system/internal/config"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/ingestion"
	"math/rand"
	"testing"
	"time"
)

func TestParseMode(t *testing.T) {
	tests := []struct {
		rawMode string
		wantErr bool
	}{
		{rawMode: "normal", wantErr: false},
		{rawMode: "warning", wantErr: false},
		{rawMode: "critical", wantErr: false},
		{rawMode: "anomaly", wantErr: false},
		{rawMode: "bad", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.rawMode, func(t *testing.T) {
			_, err := parseMode(tt.rawMode)

			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseTransport(t *testing.T) {
	tests := []struct {
		rawTransport string
		wantErr      bool
	}{
		{rawTransport: "http", wantErr: false},
		{rawTransport: "mqtt", wantErr: false},
		{rawTransport: "bad", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.rawTransport, func(t *testing.T) {
			_, err := parseTransport(tt.rawTransport)

			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSimulatorNormalReadings(t *testing.T) {
	app := newTestSimulator(SimulationModeNormal)

	readings := app.generateReadings(time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC))

	if len(readings) != 8 {
		t.Fatalf("expected 8 readings, got %d", len(readings))
	}

	pressure := findReading(t, readings, domain.ParameterPressure)
	if pressure.Value < 62 || pressure.Value > 68 {
		t.Fatalf("expected normal pressure in range 62..68, got %.2f", pressure.Value)
	}

	moisture := findReading(t, readings, domain.ParameterMoisture)
	if moisture.Value < 24 || moisture.Value > 27 {
		t.Fatalf("expected normal moisture in range 24..27, got %.2f", moisture.Value)
	}
}

func TestSimulatorWarningReadings(t *testing.T) {
	app := newTestSimulator(SimulationModeWarning)

	readings := app.generateReadings(time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC))

	pressure := findReading(t, readings, domain.ParameterPressure)
	if pressure.Value < 81 || pressure.Value > 87 {
		t.Fatalf("expected warning pressure in range 81..87, got %.2f", pressure.Value)
	}

	driveLoad := findReading(t, readings, domain.ParameterDriveLoad)
	if driveLoad.Value < 82 || driveLoad.Value > 88 {
		t.Fatalf("expected warning drive load in range 82..88, got %.2f", driveLoad.Value)
	}
}

func TestSimulatorCriticalReadings(t *testing.T) {
	app := newTestSimulator(SimulationModeCritical)

	readings := app.generateReadings(time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC))

	pressure := findReading(t, readings, domain.ParameterPressure)
	if pressure.Value < 94 || pressure.Value > 98 {
		t.Fatalf("expected critical pressure in range 94..98, got %.2f", pressure.Value)
	}

	outletTemperature := findReading(t, readings, domain.ParameterOutletTemperature)
	if outletTemperature.Value < 145 || outletTemperature.Value > 155 {
		t.Fatalf("expected critical outlet temperature in range 145..155, got %.2f", outletTemperature.Value)
	}
}

func TestSimulatorAnomalyReadingsTrend(t *testing.T) {
	app := newTestSimulator(SimulationModeAnomaly)

	firstBatch := app.generateReadings(time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC))
	secondBatch := app.generateReadings(time.Date(2026, 4, 27, 18, 0, 1, 0, time.UTC))

	firstMoisture := findReading(t, firstBatch, domain.ParameterMoisture)
	secondMoisture := findReading(t, secondBatch, domain.ParameterMoisture)

	if secondMoisture.Value >= firstMoisture.Value {
		t.Fatalf("expected moisture to decrease, first %.2f, second %.2f", firstMoisture.Value, secondMoisture.Value)
	}

	firstPressure := findReading(t, firstBatch, domain.ParameterPressure)
	secondPressure := findReading(t, secondBatch, domain.ParameterPressure)

	if secondPressure.Value <= firstPressure.Value {
		t.Fatalf("expected pressure to increase, first %.2f, second %.2f", firstPressure.Value, secondPressure.Value)
	}
}

func newTestSimulator(mode SimulationMode) *simulator {
	return &simulator{
		cfg: config.SimulatorConfig{
			SourceID: "simulator",
		},
		mode:   mode,
		random: rand.New(rand.NewSource(1)),
	}
}

func findReading(
	t *testing.T,
	readings []ingestion.TelemetryInput,
	parameterType domain.ParameterType,
) ingestion.TelemetryInput {
	t.Helper()

	for _, reading := range readings {
		if reading.ParameterType == parameterType {
			return reading
		}
	}

	t.Fatalf("reading %q not found", parameterType)
	return ingestion.TelemetryInput{}
}
