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
func CalculateQualityIndex(alerts []domain.AlertEvent) domain.QualityIndex {
	var parameterPenalty float64

	for _, alert := range alerts {
		if alert.Status == domain.AlertStatusResolved {
			continue
		}

		switch alert.Level {
		case domain.AlertLevelWarning:
			parameterPenalty += warningAlertPenalty
		case domain.AlertLevelCritical:
			parameterPenalty += criticalAlertPenalty
		}
	}

	value := clampQualityIndex(baseQualityIndex - parameterPenalty)

	return domain.QualityIndex{
		Value:            value,
		State:            domain.QualityStateFromValue(value),
		ParameterPenalty: parameterPenalty,
		AnomalyPenalty:   0,
		CalculatedAt:     time.Now().UTC(),
	}
}

func clampQualityIndex(value float64) float64 {
	if value < 0 {
		return 0
	}

	if value > 100 {
		return 100
	}

	return value
}
