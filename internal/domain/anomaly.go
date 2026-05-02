package domain

import "time"

type AnomalyID int64

type AnomalyType string

const (
	AnomalyTypeJump         AnomalyType = "jump"
	AnomalyTypeDrift        AnomalyType = "drift"
	AnomalyTypeCombinedRisk AnomalyType = "combined_risk"
)

const (
	ParameterProcessRisk ParameterType = "process"
)

type AnomalyEvent struct {
	ID            AnomalyID     `json:"id"`
	Type          AnomalyType   `json:"type"`
	ParameterType ParameterType `json:"parameterType"`
	Level         AlertLevel    `json:"level"`
	Status        AlertStatus   `json:"status"`

	Message string `json:"message"`

	CurrentValue  *float64 `json:"currentValue,omitempty"`
	PreviousValue *float64 `json:"previousValue,omitempty"`

	SourceID   SourceID  `json:"sourceId"`
	ObservedAt time.Time `json:"observedAt"`

	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	ResolvedAt *time.Time `json:"resolvedAt,omitempty"`
}
