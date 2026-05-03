package anomalies

import (
	"context"
	"extrusion-quality-system/internal/ports"
	"fmt"
	"time"

	"extrusion-quality-system/internal/domain"
)

const (
	driftWindow = 5
)

type Detector struct {
	telemetryRepository ports.TelemetryRepository
}

func NewDetector(telemetryRepository ports.TelemetryRepository) *Detector {
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
