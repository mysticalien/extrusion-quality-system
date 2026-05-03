package simulator

import (
	"time"

	"extrusion-quality-system/internal/domain"
)

type telemetryMessage struct {
	ParameterType domain.ParameterType `json:"parameterType"`
	Value         float64              `json:"value"`
	Unit          domain.Unit          `json:"unit"`
	SourceID      domain.SourceID      `json:"sourceId"`
	MeasuredAt    time.Time            `json:"measuredAt"`
}
