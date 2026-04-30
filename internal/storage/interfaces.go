package storage

import (
	"context"
	"time"

	"extrusion-quality-system/internal/domain"
)

// TelemetryRepository describes telemetry persistence operations.
type TelemetryRepository interface {
	Save(ctx context.Context, reading domain.TelemetryReading) (domain.TelemetryReading, error)
	All(ctx context.Context) ([]domain.TelemetryReading, error)
	Latest(ctx context.Context) ([]domain.TelemetryReading, error)
	HistoryByParameter(
		ctx context.Context,
		parameterType domain.ParameterType,
		from time.Time,
		to time.Time,
		limit int,
	) ([]domain.TelemetryReading, error)
}

// AlertRepository describes alert event persistence operations.
type AlertRepository interface {
	Create(ctx context.Context, alert domain.AlertEvent) (domain.AlertEvent, error)
	All(ctx context.Context) ([]domain.AlertEvent, error)
	Active(ctx context.Context) ([]domain.AlertEvent, error)
	Acknowledge(ctx context.Context, id domain.AlertID, userID *domain.UserID) (domain.AlertEvent, bool, error)
	Resolve(ctx context.Context, id domain.AlertID) (domain.AlertEvent, bool, error)
}

// QualityRepository describes quality index persistence operations.
type QualityRepository interface {
	Save(ctx context.Context, index domain.QualityIndex) (domain.QualityIndex, error)
	Latest(ctx context.Context) (domain.QualityIndex, bool, error)
	History(ctx context.Context, from time.Time, to time.Time, limit int) ([]domain.QualityIndex, error)
}
