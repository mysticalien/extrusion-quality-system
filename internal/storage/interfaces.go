package storage

import (
	"context"
	"time"

	"extrusion-quality-system/internal/domain"
)

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

type AlertRepository interface {
	Create(ctx context.Context, alert domain.AlertEvent) (domain.AlertEvent, error)
	All(ctx context.Context) ([]domain.AlertEvent, error)
	Active(ctx context.Context) ([]domain.AlertEvent, error)

	FindOpenByParameter(
		ctx context.Context,
		parameterType domain.ParameterType,
	) (domain.AlertEvent, bool, error)

	UpdateOpen(
		ctx context.Context,
		alert domain.AlertEvent,
	) (domain.AlertEvent, bool, error)

	ResolveOpenByParameter(
		ctx context.Context,
		parameterType domain.ParameterType,
	) (int64, error)

	Acknowledge(ctx context.Context, id domain.AlertID, userID *domain.UserID) (domain.AlertEvent, bool, error)
	Resolve(ctx context.Context, id domain.AlertID) (domain.AlertEvent, bool, error)
}

type QualityRepository interface {
	Save(ctx context.Context, index domain.QualityIndex) (domain.QualityIndex, error)
	Latest(ctx context.Context) (domain.QualityIndex, bool, error)
	History(ctx context.Context, from time.Time, to time.Time, limit int) ([]domain.QualityIndex, error)
}

type SetpointRepository interface {
	All(ctx context.Context) ([]domain.Setpoint, error)
	GetByParameter(ctx context.Context, parameterType domain.ParameterType) (domain.Setpoint, bool, error)
}
