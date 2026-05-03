package simulator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"extrusion-quality-system/internal/config"
)

type telemetrySender interface {
	Send(ctx context.Context, reading telemetryMessage) error
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
