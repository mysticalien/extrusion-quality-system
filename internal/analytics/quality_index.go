package analytics

import (
	"extrusion-quality-system/internal/domain"
	"time"
)

const (
	baseQualityIndex     = 100.0
	warningAlertPenalty  = 15.0
	criticalAlertPenalty = 30.0
)

// CalculateQualityIndex calculates the current extrusion quality index
// based on active and acknowledged alert events.
func CalculateQualityIndex(
	activeAlerts []domain.AlertEvent,
	anomalyGroups ...[]domain.AnomalyEvent,
) domain.QualityIndex {
	var parameterPenalty float64
	var anomalyPenalty float64

	for _, alert := range activeAlerts {
		if !isOpenStatus(alert.Status) {
			continue
		}

		switch alert.Level {
		case domain.AlertLevelWarning:
			parameterPenalty += 15
		case domain.AlertLevelCritical:
			parameterPenalty += 30
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

	value := 100 - parameterPenalty - anomalyPenalty

	if value < 0 {
		value = 0
	}

	if value > 100 {
		value = 100
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
