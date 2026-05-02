package main

import (
	"context"
	"encoding/json"
	"errors"
	"extrusion-quality-system/internal/config"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/ingestion"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type SimulationMode string

const (
	SimulationModeNormal   SimulationMode = "normal"
	SimulationModeWarning  SimulationMode = "warning"
	SimulationModeCritical SimulationMode = "critical"
	SimulationModeAnomaly  SimulationMode = "anomaly"
)

type telemetrySender interface {
	Send(ctx context.Context, reading ingestion.TelemetryInput) error
	Close()
}

type simulator struct {
	logger    *slog.Logger
	cfg       config.SimulatorConfig
	mode      SimulationMode
	sender    telemetrySender
	random    *rand.Rand
	tickCount int
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.LoadSimulator()
	if err != nil {
		logger.Error("load simulator config failed", "error", err)
		os.Exit(1)
	}

	mode, err := parseMode(cfg.Mode)
	if err != nil {
		logger.Error("invalid simulator mode", "error", err)
		os.Exit(1)
	}

	sender, err := newMQTTSender(cfg)
	if err != nil {
		logger.Error("create simulator sender failed", "error", err)
		os.Exit(1)
	}
	defer sender.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app := &simulator{
		logger: logger,
		cfg:    cfg,
		mode:   mode,
		sender: sender,
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	logger.Info(
		"starting telemetry simulator",
		"mode", mode,
		"period", cfg.Period.String(),
		"sourceId", cfg.SourceID,
	)

	if err := app.run(ctx); err != nil {
		logger.Error("simulator stopped with error", "error", err)
		os.Exit(1)
	}

	logger.Info("simulator stopped")
}

func parseMode(rawMode string) (SimulationMode, error) {
	mode := SimulationMode(strings.ToLower(strings.TrimSpace(rawMode)))

	switch mode {
	case SimulationModeNormal,
		SimulationModeWarning,
		SimulationModeCritical,
		SimulationModeAnomaly:
		return mode, nil
	default:
		return "", fmt.Errorf("unknown simulator mode %q", rawMode)
	}
}

func (s *simulator) run(ctx context.Context) error {
	if s.cfg.Period <= 0 {
		return errors.New("simulator period must be positive")
	}

	if err := s.sendBatch(ctx); err != nil {
		s.logger.Error("send telemetry batch failed", "error", err)
	}

	ticker := time.NewTicker(s.cfg.Period)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := s.sendBatch(ctx); err != nil {
				s.logger.Error("send telemetry batch failed", "error", err)
			}
		}
	}
}

func (s *simulator) sendBatch(ctx context.Context) error {
	readings := s.generateReadings(time.Now().UTC())

	var batchErr error

	for _, reading := range readings {
		if err := s.sender.Send(ctx, reading); err != nil {
			batchErr = errors.Join(batchErr, err)
			continue
		}

		s.logger.Info(
			"telemetry reading sent",
			"parameterType", reading.ParameterType,
			"value", reading.Value,
			"unit", reading.Unit,
			"mode", s.mode,
		)
	}

	return batchErr
}

func (s *simulator) generateReadings(measuredAt time.Time) []ingestion.TelemetryInput {
	s.tickCount++

	switch s.mode {
	case SimulationModeWarning:
		return s.warningReadings(measuredAt)
	case SimulationModeCritical:
		return s.criticalReadings(measuredAt)
	case SimulationModeAnomaly:
		return s.anomalyReadings(measuredAt)
	default:
		return s.normalReadings(measuredAt)
	}
}

func (s *simulator) normalReadings(measuredAt time.Time) []ingestion.TelemetryInput {
	return []ingestion.TelemetryInput{
		s.reading(domain.ParameterPressure, round2(randomInRange(s.random, 62, 68)), domain.UnitBar, measuredAt),
		s.reading(domain.ParameterMoisture, round2(randomInRange(s.random, 24, 27)), domain.UnitPercent, measuredAt),
		s.reading(domain.ParameterBarrelTemperatureZone1, round2(randomInRange(s.random, 100, 115)), domain.UnitCelsius, measuredAt),
		s.reading(domain.ParameterBarrelTemperatureZone2, round2(randomInRange(s.random, 115, 130)), domain.UnitCelsius, measuredAt),
		s.reading(domain.ParameterBarrelTemperatureZone3, round2(randomInRange(s.random, 125, 145)), domain.UnitCelsius, measuredAt),
		s.reading(domain.ParameterScrewSpeed, round2(randomInRange(s.random, 280, 360)), domain.UnitRPM, measuredAt),
		s.reading(domain.ParameterDriveLoad, round2(randomInRange(s.random, 55, 70)), domain.UnitPercent, measuredAt),
		s.reading(domain.ParameterOutletTemperature, round2(randomInRange(s.random, 105, 120)), domain.UnitCelsius, measuredAt),
	}
}

func (s *simulator) warningReadings(measuredAt time.Time) []ingestion.TelemetryInput {
	readings := s.normalReadings(measuredAt)

	replaceReading(readings, domain.ParameterPressure, round2(randomInRange(s.random, 81, 87)))
	replaceReading(readings, domain.ParameterDriveLoad, round2(randomInRange(s.random, 82, 88)))

	return readings
}

func (s *simulator) criticalReadings(measuredAt time.Time) []ingestion.TelemetryInput {
	readings := s.normalReadings(measuredAt)

	replaceReading(readings, domain.ParameterPressure, round2(randomInRange(s.random, 94, 98)))
	replaceReading(readings, domain.ParameterDriveLoad, round2(randomInRange(s.random, 93, 98)))
	replaceReading(readings, domain.ParameterBarrelTemperatureZone3, round2(randomInRange(s.random, 165, 175)))
	replaceReading(readings, domain.ParameterOutletTemperature, round2(randomInRange(s.random, 145, 155)))

	return readings
}

func (s *simulator) anomalyReadings(measuredAt time.Time) []ingestion.TelemetryInput {
	readings := s.normalReadings(measuredAt)

	tick := float64(s.tickCount)

	moisture := clamp(27-tick*0.3, 15, 27)
	pressure := clamp(55+tick*1.2, 55, 98)
	driveLoad := clamp(45+tick*1.0, 45, 96)

	replaceReading(readings, domain.ParameterMoisture, round2(moisture))
	replaceReading(readings, domain.ParameterPressure, round2(pressure))
	replaceReading(readings, domain.ParameterDriveLoad, round2(driveLoad))

	return readings
}

func (s *simulator) reading(
	parameterType domain.ParameterType,
	value float64,
	unit domain.Unit,
	measuredAt time.Time,
) ingestion.TelemetryInput {
	return ingestion.TelemetryInput{
		ParameterType: parameterType,
		Value:         value,
		Unit:          unit,
		SourceID:      domain.SourceID(s.cfg.SourceID),
		MeasuredAt:    measuredAt,
	}
}

func replaceReading(readings []ingestion.TelemetryInput, parameterType domain.ParameterType, value float64) {
	for i := range readings {
		if readings[i].ParameterType == parameterType {
			readings[i].Value = value
			return
		}
	}
}

func randomInRange(random *rand.Rand, minValue float64, maxValue float64) float64 {
	return minValue + random.Float64()*(maxValue-minValue)
}

func clamp(value float64, minValue float64, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}

	if value > maxValue {
		return maxValue
	}

	return value
}

func round2(value float64) float64 {
	return float64(int(value*100)) / 100
}

type mqttSender struct {
	client paho.Client
	topic  string
	qos    byte
}

func newMQTTSender(cfg config.SimulatorConfig) (*mqttSender, error) {
	options := paho.NewClientOptions().
		AddBroker(cfg.MQTTBrokerURL).
		SetClientID(cfg.MQTTClientID).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetConnectTimeout(cfg.RequestTimeout)

	client := paho.NewClient(options)

	token := client.Connect()
	if !token.WaitTimeout(cfg.RequestTimeout) {
		return nil, fmt.Errorf("mqtt connect timeout after %s", cfg.RequestTimeout)
	}

	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("mqtt connect: %w", err)
	}

	return &mqttSender{
		client: client,
		topic:  cfg.MQTTTopic,
		qos:    byte(cfg.MQTTQoS),
	}, nil
}

func (s *mqttSender) Send(_ context.Context, reading ingestion.TelemetryInput) error {
	body, err := json.Marshal(reading)
	if err != nil {
		return fmt.Errorf("marshal telemetry reading: %w", err)
	}

	token := s.client.Publish(s.topic, s.qos, false, body)
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("mqtt publish timeout")
	}

	if err := token.Error(); err != nil {
		return fmt.Errorf("mqtt publish: %w", err)
	}

	return nil
}

func (s *mqttSender) Close() {
	if s.client != nil && s.client.IsConnected() {
		s.client.Disconnect(250)
	}
}
