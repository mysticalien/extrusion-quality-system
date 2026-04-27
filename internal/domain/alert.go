package domain

import "time"

type SourceID string

type AlertID int64
type AlertLevel string

const (
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

type AlertStatus string

const (
	AlertStatusActive       AlertStatus = "active"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
)

// AlertEvent represents a warning or critical event detected during telemetry processing.
type AlertEvent struct {
	ID            AlertID       `json:"id"`
	ParameterType ParameterType `json:"parameterType"`
	Level         AlertLevel    `json:"level"`
	Status        AlertStatus   `json:"status"`

	Value    float64  `json:"value"`
	Unit     Unit     `json:"unit"`
	SourceID SourceID `json:"sourceId"`

	Message string `json:"message"`

	CreatedAt time.Time `json:"createdAt"`
	// AcknowledgedAt is set when an operator or technologist confirms the alert.
	AcknowledgedAt *time.Time `json:"acknowledgedAt,omitempty"`
	// AcknowledgedBy stores the user who confirmed the alert.
	AcknowledgedBy *UserID `json:"acknowledgedBy,omitempty"`
	// ResolvedAt is set when the parameter returns to an acceptable state.
	ResolvedAt *time.Time `json:"resolvedAt,omitempty"`
}
