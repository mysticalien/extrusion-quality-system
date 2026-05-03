package telemetry

import (
	"context"
	"extrusion-quality-system/internal/domain"
	"fmt"
)

func (s *Service) processAnomalies(
	ctx context.Context,
	reading domain.TelemetryReading,
) ([]domain.AnomalyEvent, error) {
	detectedAnomalies, err := s.anomalyDetector.Detect(ctx, reading)
	if err != nil {
		return nil, fmt.Errorf("detect anomalies: %w", err)
	}

	for _, anomaly := range detectedAnomalies {
		s.logger.Warn(
			"anomaly detected",
			"type", anomaly.Type,
			"parameterType", anomaly.ParameterType,
			"level", anomaly.Level,
			"message", anomaly.Message,
			"sourceId", anomaly.SourceID,
			"observedAt", anomaly.ObservedAt,
		)
	}

	if err := s.syncAnomalies(ctx, reading.ParameterType, detectedAnomalies); err != nil {
		return nil, err
	}

	activeAnomalies, err := s.anomalyRepository.Active(ctx)
	if err != nil {
		return nil, fmt.Errorf("load active anomalies: %w", err)
	}

	return activeAnomalies, nil
}

func (s *Service) syncAnomalies(
	ctx context.Context,
	currentParameter domain.ParameterType,
	detectedAnomalies []domain.AnomalyEvent,
) error {
	detected := make(map[string]struct{})

	for _, anomaly := range detectedAnomalies {
		key := anomalyKey(anomaly.Type, anomaly.ParameterType)
		detected[key] = struct{}{}

		if _, err := s.createOrUpdateOpenAnomaly(ctx, anomaly); err != nil {
			return err
		}
	}

	if _, ok := detected[anomalyKey(domain.AnomalyTypeJump, currentParameter)]; !ok {
		if _, err := s.anomalyRepository.ResolveOpenByTypeAndParameter(
			ctx,
			domain.AnomalyTypeJump,
			currentParameter,
		); err != nil {
			return fmt.Errorf("resolve jump anomaly: %w", err)
		}
	}

	if _, ok := detected[anomalyKey(domain.AnomalyTypeDrift, currentParameter)]; !ok {
		if _, err := s.anomalyRepository.ResolveOpenByTypeAndParameter(
			ctx,
			domain.AnomalyTypeDrift,
			currentParameter,
		); err != nil {
			return fmt.Errorf("resolve drift anomaly: %w", err)
		}
	}

	if _, ok := detected[anomalyKey(domain.AnomalyTypeCombinedRisk, domain.ParameterProcessRisk)]; !ok {
		if _, err := s.anomalyRepository.ResolveOpenByTypeAndParameter(
			ctx,
			domain.AnomalyTypeCombinedRisk,
			domain.ParameterProcessRisk,
		); err != nil {
			return fmt.Errorf("resolve combined risk anomaly: %w", err)
		}
	}

	return nil
}

func (s *Service) createOrUpdateOpenAnomaly(
	ctx context.Context,
	anomaly domain.AnomalyEvent,
) (domain.AnomalyEvent, error) {
	existing, found, err := s.anomalyRepository.FindOpenByTypeAndParameter(
		ctx,
		anomaly.Type,
		anomaly.ParameterType,
	)
	if err != nil {
		return domain.AnomalyEvent{}, fmt.Errorf("find open anomaly: %w", err)
	}

	if found {
		anomaly.ID = existing.ID

		updated, ok, err := s.anomalyRepository.UpdateOpen(ctx, anomaly)
		if err != nil {
			return domain.AnomalyEvent{}, fmt.Errorf("update open anomaly: %w", err)
		}

		if ok {
			return updated, nil
		}
	}

	created, err := s.anomalyRepository.Create(ctx, anomaly)
	if err != nil {
		return domain.AnomalyEvent{}, fmt.Errorf("create anomaly: %w", err)
	}

	return created, nil
}

func anomalyKey(anomalyType domain.AnomalyType, parameterType domain.ParameterType) string {
	return string(anomalyType) + ":" + string(parameterType)
}
