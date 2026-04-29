package domain

import "time"

// QualityIndexID identifies a calculated quality index value.
type QualityIndexID int64

// QualityState describes the overall extrusion process state based on the quality index.
type QualityState string

const (
	QualityStateStable   QualityState = "stable"
	QualityStateWarning  QualityState = "warning"
	QualityStateUnstable QualityState = "unstable"
	QualityStateCritical QualityState = "critical"
)

// QualityIndex represents an integral assessment of the extrusion process quality.
type QualityIndex struct {
	ID    QualityIndexID `json:"id"`
	Value float64        `json:"value"`
	State QualityState   `json:"state"`

	// ParameterPenalty is the penalty caused by deviations from technological setpoints.
	ParameterPenalty float64 `json:"parameterPenalty"`

	// AnomalyPenalty is the penalty caused by detected abnormal process behavior.
	AnomalyPenalty float64 `json:"anomalyPenalty"`

	CalculatedAt time.Time `json:"calculatedAt"`
}

// QualityStateFromValue returns the overall quality state for the given quality index value.
func QualityStateFromValue(value float64) QualityState {
	switch {
	case value >= 80:
		return QualityStateStable
	case value >= 60:
		return QualityStateWarning
	case value >= 40:
		return QualityStateUnstable
	default:
		return QualityStateCritical
	}
}
