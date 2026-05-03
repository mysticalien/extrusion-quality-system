package telemetry

import (
	"context"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/ports"
	"extrusion-quality-system/internal/usecase/anomalies"
	"extrusion-quality-system/internal/usecase/quality"
	"fmt"
	"log/slog"
)

type Option func(*Service)

func WithQualityWeights(weights quality.Weights) Option {
	return func(service *Service) {
		service.qualityWeights = weights
	}
}

func WithQualityWeightRepository(repository ports.QualityWeightRepository) Option {
	return func(service *Service) {
		service.qualityWeightRepository = repository
	}
}

type Service struct {
	logger                  *slog.Logger
	telemetryRepository     ports.TelemetryRepository
	alertRepository         ports.AlertRepository
	qualityRepository       ports.QualityRepository
	setpointRepository      ports.SetpointRepository
	anomalyRepository       ports.AnomalyRepository
	anomalyDetector         *anomalies.Detector
	qualityWeightRepository ports.QualityWeightRepository
	qualityWeights          quality.Weights
}

// NewService creates telemetry usecase service.
func NewService(
	logger *slog.Logger,
	telemetryRepository ports.TelemetryRepository,
	alertRepository ports.AlertRepository,
	qualityRepository ports.QualityRepository,
	setpointRepository ports.SetpointRepository,
	anomalyRepository ports.AnomalyRepository,
	options ...Option,
) *Service {
	service := &Service{
		logger:              logger,
		telemetryRepository: telemetryRepository,
		alertRepository:     alertRepository,
		qualityRepository:   qualityRepository,
		setpointRepository:  setpointRepository,
		anomalyRepository:   anomalyRepository,
		anomalyDetector:     anomalies.NewDetector(telemetryRepository),
		qualityWeights:      quality.DefaultWeights(),
	}

	for _, option := range options {
		option(service)
	}

	return service
}

func (s *Service) validate(ctx context.Context, input Input) (domain.Setpoint, error) {
	setpoint, found, err := s.setpointRepository.GetByParameter(ctx, input.ParameterType)
	if err != nil {
		return domain.Setpoint{}, fmt.Errorf("load setpoint by parameter: %w", err)
	}

	if !found {
		return domain.Setpoint{}, ValidationError{Message: "unknown parameterType"}
	}

	if input.Unit == "" {
		return domain.Setpoint{}, ValidationError{Message: "unit is required"}
	}

	if input.Unit != setpoint.Unit {
		return domain.Setpoint{}, ValidationError{Message: "unit does not match parameterType"}
	}

	if input.SourceID == "" {
		return domain.Setpoint{}, ValidationError{Message: "sourceId is required"}
	}

	if input.MeasuredAt.IsZero() {
		return domain.Setpoint{}, ValidationError{Message: "measuredAt is required"}
	}

	return setpoint, nil
}
