package telemetry

import (
	"context"
	"fmt"
	"time"

	"extrusion-quality-system/internal/domain"
)

// Process validates, stores and analytically processes one telemetry reading.
// It also prevents duplicate active alerts for the same parameter.
func (s *Service) Process(ctx context.Context, input Input) (Result, error) {
	setpoint, err := s.validate(ctx, input)
	if err != nil {
		return Result{}, err
	}

	s.logger.Info(
		"telemetry received",
		"parameterType", input.ParameterType,
		"value", input.Value,
		"unit", input.Unit,
		"sourceId", input.SourceID,
		"measuredAt", input.MeasuredAt,
	)

	reading, err := s.saveTelemetryReading(ctx, input)
	if err != nil {
		return Result{}, err
	}

	state := setpoint.Evaluate(reading.Value)

	alertResult, err := s.processAlerts(ctx, reading, state)
	if err != nil {
		return Result{}, err
	}

	activeAlerts, err := s.alertRepository.Active(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("load active alerts: %w", err)
	}

	activeAnomalies, err := s.processAnomalies(ctx, reading)
	if err != nil {
		return Result{}, err
	}

	qualityIndex, err := s.calculateAndSaveQualityIndex(ctx, activeAlerts, activeAnomalies)
	if err != nil {
		return Result{}, err
	}

	s.logger.Info(
		"telemetry processed",
		"parameterType", reading.ParameterType,
		"value", reading.Value,
		"unit", reading.Unit,
		"sourceId", reading.SourceID,
		"state", state,
		"alertCreated", alertResult.Created,
		"alertUpdated", alertResult.Updated,
		"resolvedAlerts", alertResult.ResolvedCount,
		"qualityIndex", qualityIndex.Value,
		"qualityState", qualityIndex.State,
	)

	return buildResult(reading, state, alertResult, qualityIndex), nil
}

func (s *Service) saveTelemetryReading(
	ctx context.Context,
	input Input,
) (domain.TelemetryReading, error) {
	reading := domain.TelemetryReading{
		ParameterType: input.ParameterType,
		Value:         input.Value,
		Unit:          input.Unit,
		SourceID:      input.SourceID,
		MeasuredAt:    input.MeasuredAt,
		CreatedAt:     time.Now().UTC(),
	}

	reading, err := s.telemetryRepository.Save(ctx, reading)
	if err != nil {
		return domain.TelemetryReading{}, fmt.Errorf("save telemetry reading: %w", err)
	}

	return reading, nil
}
