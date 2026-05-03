package telemetry

import (
	"extrusion-quality-system/internal/domain"
	"time"
)

// Input describes telemetry data received from external sources.
// It is used by HTTP, MQTT, Kafka and simulator before the value is stored.
type Input struct {
	ParameterType domain.ParameterType `json:"parameterType"`
	Value         float64              `json:"value"`
	Unit          domain.Unit          `json:"unit"`
	SourceID      domain.SourceID      `json:"sourceId"`
	MeasuredAt    time.Time            `json:"measuredAt"`
}
