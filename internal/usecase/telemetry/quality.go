package telemetry

import (
	"context"
	"fmt"

	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/usecase/quality"
)

func (s *Service) calculateAndSaveQualityIndex(
	ctx context.Context,
	activeAlerts []domain.AlertEvent,
	activeAnomalies []domain.AnomalyEvent,
) (domain.QualityIndex, error) {
	qualityWeights, err := s.loadQualityWeights(ctx)
	if err != nil {
		return domain.QualityIndex{}, err
	}

	qualityIndex := quality.CalculateIndex(
		activeAlerts,
		qualityWeights,
		activeAnomalies,
	)

	s.logger.Info(
		"quality index calculated",
		"value", qualityIndex.Value,
		"state", qualityIndex.State,
		"parameterPenalty", qualityIndex.ParameterPenalty,
		"anomalyPenalty", qualityIndex.AnomalyPenalty,
	)

	qualityIndex, err = s.qualityRepository.Save(ctx, qualityIndex)
	if err != nil {
		return domain.QualityIndex{}, fmt.Errorf("save quality index: %w", err)
	}

	return qualityIndex, nil
}

func (s *Service) loadQualityWeights(ctx context.Context) (quality.Weights, error) {
	if s.qualityWeightRepository == nil {
		return s.qualityWeights, nil
	}

	items, err := s.qualityWeightRepository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("load quality weights: %w", err)
	}

	if len(items) == 0 {
		return s.qualityWeights, nil
	}

	return quality.WeightsFromDomain(items), nil
}
