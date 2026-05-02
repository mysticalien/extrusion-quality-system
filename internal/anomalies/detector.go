package anomalies

import (
	"context"
	"fmt"
	"math"
	"time"

	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/storage"
)

const (
	driftWindow = 5
)

type Detector struct {
	telemetryRepository storage.TelemetryRepository
}

func NewDetector(telemetryRepository storage.TelemetryRepository) *Detector {
	return &Detector{
		telemetryRepository: telemetryRepository,
	}
}

func (d *Detector) Detect(
	ctx context.Context,
	current domain.TelemetryReading,
) ([]domain.AnomalyEvent, error) {
	anomalies := make([]domain.AnomalyEvent, 0)

	parameterHistory, err := d.telemetryRepository.HistoryByParameter(
		ctx,
		current.ParameterType,
		time.Time{},
		time.Time{},
		driftWindow,
	)
	if err != nil {
		return nil, fmt.Errorf("load parameter history for anomaly detection: %w", err)
	}

	if anomaly, ok := detectJump(current, parameterHistory); ok {
		anomalies = append(anomalies, anomaly)
	}

	if anomaly, ok := detectDrift(current, parameterHistory); ok {
		anomalies = append(anomalies, anomaly)
	}

	combinedRisk, err := d.detectCombinedRisk(ctx, current)
	if err != nil {
		return nil, err
	}

	if combinedRisk != nil {
		anomalies = append(anomalies, *combinedRisk)
	}

	return anomalies, nil
}

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

func (d *Detector) detectCombinedRisk(
	ctx context.Context,
	current domain.TelemetryReading,
) (*domain.AnomalyEvent, error) {
	pressureHistory, err := d.telemetryRepository.HistoryByParameter(
		ctx,
		domain.ParameterPressure,
		time.Time{},
		time.Time{},
		driftWindow,
	)
	if err != nil {
		return nil, fmt.Errorf("load pressure history: %w", err)
	}

	moistureHistory, err := d.telemetryRepository.HistoryByParameter(
		ctx,
		domain.ParameterMoisture,
		time.Time{},
		time.Time{},
		driftWindow,
	)
	if err != nil {
		return nil, fmt.Errorf("load moisture history: %w", err)
	}

	driveLoadHistory, err := d.telemetryRepository.HistoryByParameter(
		ctx,
		domain.ParameterDriveLoad,
		time.Time{},
		time.Time{},
		driftWindow,
	)
	if err != nil {
		return nil, fmt.Errorf("load drive load history: %w", err)
	}

	if len(pressureHistory) < driftWindow ||
		len(moistureHistory) < driftWindow ||
		len(driveLoadHistory) < driftWindow {
		return nil, nil
	}

	pressureRising := isNonDecreasing(pressureHistory) &&
		pressureHistory[len(pressureHistory)-1].Value-pressureHistory[0].Value >= 5

	moistureFalling := isNonIncreasing(moistureHistory) &&
		moistureHistory[0].Value-moistureHistory[len(moistureHistory)-1].Value >= 1.5

	driveLoadRising := isNonDecreasing(driveLoadHistory) &&
		driveLoadHistory[len(driveLoadHistory)-1].Value-driveLoadHistory[0].Value >= 5

	if !pressureRising || !moistureFalling || !driveLoadRising {
		return nil, nil
	}

	return &domain.AnomalyEvent{
		Type:          domain.AnomalyTypeCombinedRisk,
		ParameterType: domain.ParameterProcessRisk,
		Level:         domain.AlertLevelCritical,
		Status:        domain.AlertStatusActive,
		Message:       "combined extrusion instability risk: moisture is falling while pressure and drive load are rising",
		SourceID:      current.SourceID,
		ObservedAt:    current.MeasuredAt,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}, nil
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
