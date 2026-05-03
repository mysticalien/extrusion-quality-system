package anomalies

import (
	"extrusion-quality-system/internal/domain"
	"fmt"
	"math"
	"time"
)

func detectDrift(
	current domain.TelemetryReading,
	history []domain.TelemetryReading,
) (domain.AnomalyEvent, bool) {
	if len(history) < driftWindow {
		return domain.AnomalyEvent{}, false
	}

	trend := expectedDriftTrend(current.ParameterType)
	if trend == "" {
		return domain.AnomalyEvent{}, false
	}

	first := history[0]
	last := history[len(history)-1]
	totalDelta := last.Value - first.Value

	isDrift := false

	switch trend {
	case "rising":
		isDrift = isNonDecreasing(history) && totalDelta >= driftThreshold(current.ParameterType)
	case "falling":
		isDrift = isNonIncreasing(history) && math.Abs(totalDelta) >= driftThreshold(current.ParameterType)
	}

	if !isDrift {
		return domain.AnomalyEvent{}, false
	}

	currentValue := current.Value
	previousValue := first.Value

	return domain.AnomalyEvent{
		Type:          domain.AnomalyTypeDrift,
		ParameterType: current.ParameterType,
		Level:         domain.AlertLevelWarning,
		Status:        domain.AlertStatusActive,
		Message: fmt.Sprintf(
			"drift detected for %s over last %d measurements",
			current.ParameterType,
			len(history),
		),
		CurrentValue:  &currentValue,
		PreviousValue: &previousValue,
		SourceID:      current.SourceID,
		ObservedAt:    current.MeasuredAt,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}, true
}

func driftThreshold(parameterType domain.ParameterType) float64 {
	switch parameterType {
	case domain.ParameterPressure:
		return 6
	case domain.ParameterMoisture:
		return 1.5
	case domain.ParameterDriveLoad:
		return 5
	case domain.ParameterBarrelTemperatureZone1,
		domain.ParameterBarrelTemperatureZone2,
		domain.ParameterBarrelTemperatureZone3,
		domain.ParameterOutletTemperature:
		return 8
	default:
		return 0
	}
}

func expectedDriftTrend(parameterType domain.ParameterType) string {
	switch parameterType {
	case domain.ParameterPressure,
		domain.ParameterDriveLoad,
		domain.ParameterBarrelTemperatureZone1,
		domain.ParameterBarrelTemperatureZone2,
		domain.ParameterBarrelTemperatureZone3,
		domain.ParameterOutletTemperature:
		return "rising"
	case domain.ParameterMoisture:
		return "falling"
	default:
		return ""
	}
}

func isNonDecreasing(readings []domain.TelemetryReading) bool {
	for i := 1; i < len(readings); i++ {
		if readings[i].Value < readings[i-1].Value {
			return false
		}
	}

	return true
}

func isNonIncreasing(readings []domain.TelemetryReading) bool {
	for i := 1; i < len(readings); i++ {
		if readings[i].Value > readings[i-1].Value {
			return false
		}
	}

	return true
}
