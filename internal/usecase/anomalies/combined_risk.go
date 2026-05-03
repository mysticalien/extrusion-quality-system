package anomalies

import (
	"context"
	"extrusion-quality-system/internal/domain"
	"fmt"
	"time"
)

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
