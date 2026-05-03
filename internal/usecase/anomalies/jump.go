package anomalies

import (
	"extrusion-quality-system/internal/domain"
	"fmt"
	"math"
	"time"
)

func detectJump(
	current domain.TelemetryReading,
	history []domain.TelemetryReading,
) (domain.AnomalyEvent, bool) {
	if len(history) < 2 {
		return domain.AnomalyEvent{}, false
	}

	previous := history[len(history)-2]

	delta := current.Value - previous.Value
	threshold := jumpThreshold(current.ParameterType)

	if threshold <= 0 || !isJumpInExpectedDirection(current.ParameterType, delta, threshold) {
		return domain.AnomalyEvent{}, false
	}

	currentValue := current.Value
	previousValue := previous.Value

	return domain.AnomalyEvent{
		Type:          domain.AnomalyTypeJump,
		ParameterType: current.ParameterType,
		Level:         domain.AlertLevelWarning,
		Status:        domain.AlertStatusActive,
		Message: fmt.Sprintf(
			"sharp jump detected for %s: %.2f -> %.2f %s",
			current.ParameterType,
			previous.Value,
			current.Value,
			current.Unit,
		),
		CurrentValue:  &currentValue,
		PreviousValue: &previousValue,
		SourceID:      current.SourceID,
		ObservedAt:    current.MeasuredAt,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}, true
}

func jumpThreshold(parameterType domain.ParameterType) float64 {
	switch parameterType {
	case domain.ParameterPressure:
		return 10
	case domain.ParameterMoisture:
		return 3
	case domain.ParameterDriveLoad:
		return 15
	case domain.ParameterScrewSpeed:
		return 80
	case domain.ParameterBarrelTemperatureZone1,
		domain.ParameterBarrelTemperatureZone2,
		domain.ParameterBarrelTemperatureZone3,
		domain.ParameterOutletTemperature:
		return 15
	default:
		return 0
	}
}

func isJumpInExpectedDirection(
	parameterType domain.ParameterType,
	delta float64,
	threshold float64,
) bool {
	switch parameterType {
	case domain.ParameterPressure,
		domain.ParameterDriveLoad,
		domain.ParameterBarrelTemperatureZone1,
		domain.ParameterBarrelTemperatureZone2,
		domain.ParameterBarrelTemperatureZone3,
		domain.ParameterOutletTemperature:
		return delta >= threshold

	case domain.ParameterMoisture:
		return -delta >= threshold

	case domain.ParameterScrewSpeed:
		return math.Abs(delta) >= threshold

	default:
		return false
	}
}
