package analytics

import (
	"time"

	"extrusion-quality-system/internal/domain"
)

const (
	baseQualityIndex     = 100.0
	warningAlertPenalty  = 15.0
	criticalAlertPenalty = 30.0
)

// CalculateQualityIndex calculates the current extrusion quality index
// based on active and acknowledged alert and anomaly events.
//
// Parameter weights affect parameter penalties:
// a parameter with a higher weight decreases quality index stronger.
func CalculateQualityIndex(
	activeAlerts []domain.AlertEvent,
	qualityWeights QualityWeights,
	anomalyGroups ...[]domain.AnomalyEvent,
) domain.QualityIndex {
	var parameterPenalty float64
	var anomalyPenalty float64

	if qualityWeights == nil {
		qualityWeights = DefaultQualityWeights()
	}

	for _, alert := range activeAlerts {
		if !isOpenStatus(alert.Status) {
			continue
		}

		weight := qualityWeights.WeightFor(alert.ParameterType)

		switch alert.Level {
		case domain.AlertLevelWarning:
			parameterPenalty += warningAlertPenalty * weight

		case domain.AlertLevelCritical:
			parameterPenalty += criticalAlertPenalty * weight
		}
	}

	if len(anomalyGroups) > 0 {
		for _, anomaly := range anomalyGroups[0] {
			if !isOpenStatus(anomaly.Status) {
				continue
			}

			switch anomaly.Type {
			case domain.AnomalyTypeJump:
				anomalyPenalty += 10

			case domain.AnomalyTypeDrift:
				anomalyPenalty += 15

			case domain.AnomalyTypeCombinedRisk:
				anomalyPenalty += 25
			}
		}
	}

	value := baseQualityIndex - parameterPenalty - anomalyPenalty

	if value < 0 {
		value = 0
	}

	if value > baseQualityIndex {
		value = baseQualityIndex
	}

	return domain.QualityIndex{
		Value:            value,
		State:            domain.QualityStateFromValue(value),
		ParameterPenalty: parameterPenalty,
		AnomalyPenalty:   anomalyPenalty,
		CalculatedAt:     time.Now().UTC(),
	}
}

func isOpenStatus(status domain.AlertStatus) bool {
	return status == domain.AlertStatusActive ||
		status == domain.AlertStatusAcknowledged
}
