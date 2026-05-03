package ports

import (
	"context"
	"time"

	"extrusion-quality-system/internal/domain"
)

type TelemetryRepository interface {
	Save(ctx context.Context, reading domain.TelemetryReading) (domain.TelemetryReading, error)
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
	Update(
		ctx context.Context,
		id int64,
		update domain.SetpointUpdate,
	) (domain.Setpoint, bool, error)
}

type AnomalyRepository interface {
	Create(ctx context.Context, anomaly domain.AnomalyEvent) (domain.AnomalyEvent, error)
	All(ctx context.Context) ([]domain.AnomalyEvent, error)
	Active(ctx context.Context) ([]domain.AnomalyEvent, error)

	FindOpenByTypeAndParameter(
		ctx context.Context,
		anomalyType domain.AnomalyType,
		parameterType domain.ParameterType,
	) (domain.AnomalyEvent, bool, error)

	UpdateOpen(
		ctx context.Context,
		anomaly domain.AnomalyEvent,
	) (domain.AnomalyEvent, bool, error)

	ResolveOpenByTypeAndParameter(
		ctx context.Context,
		anomalyType domain.AnomalyType,
		parameterType domain.ParameterType,
	) (int64, error)
}

type UserRepository interface {
	All(ctx context.Context) ([]domain.User, error)
	FindByUsername(ctx context.Context, username string) (domain.User, bool, error)
	FindByID(ctx context.Context, id domain.UserID) (domain.User, bool, error)
	Create(ctx context.Context, user domain.User) (domain.User, error)
	UpdateRole(ctx context.Context, id domain.UserID, role domain.UserRole) (domain.User, bool, error)
	UpdatePassword(ctx context.Context, id domain.UserID, passwordHash string) (domain.User, bool, error)
	SetActive(ctx context.Context, id domain.UserID, isActive bool) (domain.User, bool, error)
}

type QualityWeightRepository interface {
	List(ctx context.Context) ([]domain.QualityWeight, error)
	Update(
		ctx context.Context,
		id domain.QualityWeightID,
		update domain.QualityWeightUpdate,
		updatedBy string,
	) (domain.QualityWeight, bool, error)
}
