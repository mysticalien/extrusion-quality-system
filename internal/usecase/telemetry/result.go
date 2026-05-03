package telemetry

import (
	"extrusion-quality-system/internal/domain"
	"time"
)

// Result describes telemetry processing result.
// Result describes telemetry processing result.
type Result struct {
	Accepted       bool                  `json:"accepted"`
	ParameterType  domain.ParameterType  `json:"parameterType"`
	Value          float64               `json:"value"`
	Unit           domain.Unit           `json:"unit"`
	SourceID       domain.SourceID       `json:"sourceId"`
	MeasuredAt     time.Time             `json:"measuredAt"`
	State          domain.ParameterState `json:"state"`
	AlertCreated   bool                  `json:"alertCreated"`
	AlertUpdated   bool                  `json:"alertUpdated,omitempty"`
	ResolvedAlerts int64                 `json:"resolvedAlerts,omitempty"`
	AlertID        *domain.AlertID       `json:"alertId,omitempty"`
	AlertLevel     *domain.AlertLevel    `json:"alertLevel,omitempty"`
	QualityIndex   float64               `json:"qualityIndex"`
	QualityState   domain.QualityState   `json:"qualityState"`
}

func buildResult(
	reading domain.TelemetryReading,
	state domain.ParameterState,
	alertResult alertProcessingResult,
	qualityIndex domain.QualityIndex,
) Result {
	return Result{
		Accepted:       true,
		ParameterType:  reading.ParameterType,
		Value:          reading.Value,
		Unit:           reading.Unit,
		SourceID:       reading.SourceID,
		MeasuredAt:     reading.MeasuredAt,
		State:          state,
		AlertCreated:   alertResult.Created,
		AlertUpdated:   alertResult.Updated,
		ResolvedAlerts: alertResult.ResolvedCount,
		AlertID:        alertResult.ID,
		AlertLevel:     alertResult.Level,
		QualityIndex:   qualityIndex.Value,
		QualityState:   qualityIndex.State,
	}
}
