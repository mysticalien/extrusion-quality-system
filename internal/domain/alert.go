package domain

import "time"

// AlertID identifies an alert event.
type AlertID int64

// AlertLevel describes the severity of an alert event.
type AlertLevel string

const (
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// AlertStatus describes the lifecycle state of an alert event.
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

// Acknowledge marks the alert as acknowledged by a user.
func (a *AlertEvent) Acknowledge(userID *UserID, acknowledgedAt time.Time) {
	a.Status = AlertStatusAcknowledged
	a.AcknowledgedAt = &acknowledgedAt

	if userID != nil {
		id := *userID
		a.AcknowledgedBy = &id
	}
}

// Resolve marks the alert as resolved.
func (a *AlertEvent) Resolve(resolvedAt time.Time) {
	a.Status = AlertStatusResolved
	a.ResolvedAt = &resolvedAt
}
