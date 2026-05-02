package simulator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"extrusion-quality-system/internal/config"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/ingestion"
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

type app struct {
	logger    *slog.Logger
	cfg       config.SimulatorConfig
	mode      SimulationMode
	sender    telemetrySender
	random    *rand.Rand
	tickCount int
}

func Run(ctx context.Context, logger *slog.Logger, cfg config.SimulatorConfig) error {
	mode, err := parseMode(cfg.Mode)
	if err != nil {
		return err
	}

	sender, err := newMQTTSender(cfg)
	if err != nil {
		return fmt.Errorf("create simulator sender: %w", err)
	}
	defer sender.Close()

	simulator := &app{
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
		"mqttBrokerUrl", cfg.MQTTBrokerURL,
		"mqttTopic", cfg.MQTTTopic,
	)

	if err := simulator.run(ctx); err != nil {
		return err
	}

	logger.Info("simulator stopped")
	return nil
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

func (a *app) run(ctx context.Context) error {
	if a.cfg.Period <= 0 {
		return errors.New("simulator period must be positive")
	}

	if err := a.sendBatch(ctx); err != nil {
		a.logger.Error("send telemetry batch failed", "error", err)
	}

	ticker := time.NewTicker(a.cfg.Period)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			if err := a.sendBatch(ctx); err != nil {
				a.logger.Error("send telemetry batch failed", "error", err)
			}
		}
	}
}

func (a *app) sendBatch(ctx context.Context) error {
	readings := a.generateReadings(time.Now().UTC())

	var batchErr error

	for _, reading := range readings {
		if err := a.sender.Send(ctx, reading); err != nil {
			batchErr = errors.Join(batchErr, err)
			continue
		}

		a.logger.Debug(
			"telemetry reading sent",
			"parameterType", reading.ParameterType,
			"value", reading.Value,
			"unit", reading.Unit,
			"mode", a.mode,
		)
	}

	return batchErr
}

func (a *app) generateReadings(measuredAt time.Time) []ingestion.TelemetryInput {
	a.tickCount++

	switch a.mode {
	case SimulationModeWarning:
		return a.warningReadings(measuredAt)
	case SimulationModeCritical:
		return a.criticalReadings(measuredAt)
	case SimulationModeAnomaly:
		return a.anomalyReadings(measuredAt)
	default:
		return a.normalReadings(measuredAt)
	}
}

func (a *app) normalReadings(measuredAt time.Time) []ingestion.TelemetryInput {
	return []ingestion.TelemetryInput{
		a.reading(domain.ParameterPressure, round2(randomInRange(a.random, 62, 68)), domain.UnitBar, measuredAt),
		a.reading(domain.ParameterMoisture, round2(randomInRange(a.random, 24, 27)), domain.UnitPercent, measuredAt),
		a.reading(domain.ParameterBarrelTemperatureZone1, round2(randomInRange(a.random, 100, 115)), domain.UnitCelsius, measuredAt),
		a.reading(domain.ParameterBarrelTemperatureZone2, round2(randomInRange(a.random, 115, 130)), domain.UnitCelsius, measuredAt),
		a.reading(domain.ParameterBarrelTemperatureZone3, round2(randomInRange(a.random, 125, 145)), domain.UnitCelsius, measuredAt),
		a.reading(domain.ParameterScrewSpeed, round2(randomInRange(a.random, 280, 360)), domain.UnitRPM, measuredAt),
		a.reading(domain.ParameterDriveLoad, round2(randomInRange(a.random, 55, 70)), domain.UnitPercent, measuredAt),
		a.reading(domain.ParameterOutletTemperature, round2(randomInRange(a.random, 105, 120)), domain.UnitCelsius, measuredAt),
	}
}

func (a *app) warningReadings(measuredAt time.Time) []ingestion.TelemetryInput {
	readings := a.normalReadings(measuredAt)

	replaceReading(readings, domain.ParameterPressure, round2(randomInRange(a.random, 81, 87)))
	replaceReading(readings, domain.ParameterDriveLoad, round2(randomInRange(a.random, 82, 88)))

	return readings
}

func (a *app) criticalReadings(measuredAt time.Time) []ingestion.TelemetryInput {
	readings := a.normalReadings(measuredAt)

	replaceReading(readings, domain.ParameterPressure, round2(randomInRange(a.random, 94, 98)))
	replaceReading(readings, domain.ParameterDriveLoad, round2(randomInRange(a.random, 93, 98)))
	replaceReading(readings, domain.ParameterBarrelTemperatureZone3, round2(randomInRange(a.random, 165, 175)))
	replaceReading(readings, domain.ParameterOutletTemperature, round2(randomInRange(a.random, 145, 155)))

	return readings
}

func (a *app) anomalyReadings(measuredAt time.Time) []ingestion.TelemetryInput {
	readings := a.normalReadings(measuredAt)

	tick := float64(a.tickCount)

	moisture := clamp(27-tick*0.3, 15, 27)
	pressure := clamp(55+tick*1.2, 55, 98)
	driveLoad := clamp(45+tick*1.0, 45, 96)

	replaceReading(readings, domain.ParameterMoisture, round2(moisture))
	replaceReading(readings, domain.ParameterPressure, round2(pressure))
	replaceReading(readings, domain.ParameterDriveLoad, round2(driveLoad))

	return readings
}

func (a *app) reading(
	parameterType domain.ParameterType,
	value float64,
	unit domain.Unit,
	measuredAt time.Time,
) ingestion.TelemetryInput {
	return ingestion.TelemetryInput{
		ParameterType: parameterType,
		Value:         value,
		Unit:          unit,
		SourceID:      domain.SourceID(a.cfg.SourceID),
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
		return errors.New("mqtt publish timeout")
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
