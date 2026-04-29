package storage

import "extrusion-quality-system/internal/domain"

// TelemetryStore describes telemetry persistence operations.
type TelemetryStore interface {
	Save(reading domain.TelemetryReading) (domain.TelemetryReading, error)
	All() ([]domain.TelemetryReading, error)
}

// AlertStore describes alert event persistence operations.
type AlertStore interface {
	Create(alert domain.AlertEvent) (domain.AlertEvent, error)
	All() ([]domain.AlertEvent, error)
	Active() ([]domain.AlertEvent, error)
	Acknowledge(id domain.AlertID, userID *domain.UserID) (domain.AlertEvent, bool, error)
	Resolve(id domain.AlertID) (domain.AlertEvent, bool, error)
}

// QualityStore describes quality index persistence operations.
type QualityStore interface {
	Save(index domain.QualityIndex) (domain.QualityIndex, error)
	Latest() (domain.QualityIndex, bool, error)
}
