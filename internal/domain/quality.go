package domain

import "time"

type QualityIndexID int64

type QualityState string

const (
	QualityStateStable   QualityState = "stable"
	QualityStateUnstable QualityState = "unstable"
	QualityStateWarning  QualityState = "warning"
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

//
//func NewQualityIndex(value, parameterPenalty, anomalyPenalty float64, calculatedAt time.Time) QualityIndex {
//	value = ClampQualityIndex(value)
//
//	return QualityIndex{
//		Value:            value,
//		State:            QualityStateFromValue(value),
//		ParameterPenalty: parameterPenalty,
//		AnomalyPenalty:   anomalyPenalty,
//		CalculatedAt:     calculatedAt,
//	}
//}
//
//func ClampQualityIndex(value float64) float64 {
//	if value < 0 {
//		return 0
//	}
//
//	if value > 100 {
//		return 100
//	}
//
//	return value
//}
//
//func QualityStateFromValue(value float64) QualityState {
//	switch {
//	case value >= 80:
//		return QualityStateStable
//	case value >= 60:
//		return QualityStateUnstable
//	case value >= 40:
//		return QualityStateWarning
//	default:
//		return QualityStateCritical
//	}
//}
