package postgres

import (
	"context"
	"time"

	"extrusion-quality-system/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AlertRepository stores alert events in PostgreSQL.
type AlertRepository struct {
	pool *pgxpool.Pool
}

// NewAlertRepository creates a PostgreSQL alert repository.
func NewAlertRepository(pool *pgxpool.Pool) *AlertRepository {
	return &AlertRepository{
		pool: pool,
	}
}

// Create stores an alert event in PostgreSQL.
func (r *AlertRepository) Create(
	ctx context.Context,
	alert domain.AlertEvent,
) (domain.AlertEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if alert.Status == "" {
		alert.Status = domain.AlertStatusActive
	}

	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = time.Now().UTC()
	}

	err := r.pool.QueryRow(
		ctx,
		`
		INSERT INTO alert_events (
			parameter_type,
			level,
			status,
			value,
			unit,
			source_id,
			message,
			created_at,
			acknowledged_at,
			acknowledged_by,
			resolved_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
		`,
		alert.ParameterType,
		alert.Level,
		alert.Status,
		alert.Value,
		alert.Unit,
		alert.SourceID,
		alert.Message,
		alert.CreatedAt,
		alert.AcknowledgedAt,
		alert.AcknowledgedBy,
		alert.ResolvedAt,
	).Scan(&alert.ID)

	if err != nil {
		return domain.AlertEvent{}, err
	}

	return alert, nil
}

// All returns all alert events ordered by creation time.
func (r *AlertRepository) All(ctx context.Context) ([]domain.AlertEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := r.pool.Query(
		ctx,
		`
		SELECT
			id,
			parameter_type,
			level,
			status,
			value,
			unit,
			source_id,
			message,
			created_at,
			acknowledged_at,
			acknowledged_by,
			resolved_at
		FROM alert_events
		ORDER BY created_at DESC, id DESC
		`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAlerts(rows)
}

// Active returns the latest active or acknowledged alert for each parameter.
func (r *AlertRepository) Active(ctx context.Context) ([]domain.AlertEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := r.pool.Query(
		ctx,
		`
		SELECT
			id,
			parameter_type,
			level,
			status,
			value,
			unit,
			source_id,
			message,
			created_at,
			acknowledged_at,
			acknowledged_by,
			resolved_at
		FROM (
			SELECT DISTINCT ON (parameter_type)
				id,
				parameter_type,
				level,
				status,
				value,
				unit,
				source_id,
				message,
				created_at,
				acknowledged_at,
				acknowledged_by,
				resolved_at
			FROM alert_events
			WHERE status IN ($1, $2)
			ORDER BY parameter_type, created_at DESC, id DESC
		) latest_open_alerts
		ORDER BY created_at DESC, id DESC
		`,
		domain.AlertStatusActive,
		domain.AlertStatusAcknowledged,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAlerts(rows)
}

// FindOpenByParameter returns the latest open alert for the parameter.
func (r *AlertRepository) FindOpenByParameter(
	ctx context.Context,
	parameterType domain.ParameterType,
) (domain.AlertEvent, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	alert, err := scanAlertRow(
		r.pool.QueryRow(
			ctx,
			`
			SELECT
				id,
				parameter_type,
				level,
				status,
				value,
				unit,
				source_id,
				message,
				created_at,
				acknowledged_at,
				acknowledged_by,
				resolved_at
			FROM alert_events
			WHERE parameter_type = $1
			  AND status IN ($2, $3)
			ORDER BY created_at DESC, id DESC
			LIMIT 1
			`,
			parameterType,
			domain.AlertStatusActive,
			domain.AlertStatusAcknowledged,
		),
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.AlertEvent{}, false, nil
		}

		return domain.AlertEvent{}, false, err
	}

	return alert, true, nil
}

// UpdateOpen updates an existing open alert without creating a duplicate.
func (r *AlertRepository) UpdateOpen(
	ctx context.Context,
	alert domain.AlertEvent,
) (domain.AlertEvent, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	updatedAlert, err := scanAlertRow(
		r.pool.QueryRow(
			ctx,
			`
			UPDATE alert_events
			SET
				level = $2,
				value = $3,
				unit = $4,
				source_id = $5,
				message = $6
			WHERE id = $1
			  AND status IN ($7, $8)
			RETURNING
				id,
				parameter_type,
				level,
				status,
				value,
				unit,
				source_id,
				message,
				created_at,
				acknowledged_at,
				acknowledged_by,
				resolved_at
			`,
			alert.ID,
			alert.Level,
			alert.Value,
			alert.Unit,
			alert.SourceID,
			alert.Message,
			domain.AlertStatusActive,
			domain.AlertStatusAcknowledged,
		),
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.AlertEvent{}, false, nil
		}

		return domain.AlertEvent{}, false, err
	}

	return updatedAlert, true, nil
}

// ResolveOpenByParameter resolves all open alerts for the given parameter.
func (r *AlertRepository) ResolveOpenByParameter(
	ctx context.Context,
	parameterType domain.ParameterType,
) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	commandTag, err := r.pool.Exec(
		ctx,
		`
		UPDATE alert_events
		SET
			status = $2,
			resolved_at = now()
		WHERE parameter_type = $1
		  AND status IN ($3, $4)
		`,
		parameterType,
		domain.AlertStatusResolved,
		domain.AlertStatusActive,
		domain.AlertStatusAcknowledged,
	)
	if err != nil {
		return 0, err
	}

	return commandTag.RowsAffected(), nil
}

// Acknowledge marks an alert event as acknowledged.
func (r *AlertRepository) Acknowledge(
	ctx context.Context,
	id domain.AlertID,
	userID *domain.UserID,
) (domain.AlertEvent, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var acknowledgedBy any
	if userID != nil {
		acknowledgedBy = *userID
	}

	alert, err := scanAlertRow(
		r.pool.QueryRow(
			ctx,
			`
			UPDATE alert_events
			SET
				status = $2,
				acknowledged_at = now(),
				acknowledged_by = $3
			WHERE id = $1
			RETURNING
				id,
				parameter_type,
				level,
				status,
				value,
				unit,
				source_id,
				message,
				created_at,
				acknowledged_at,
				acknowledged_by,
				resolved_at
			`,
			id,
			domain.AlertStatusAcknowledged,
			acknowledgedBy,
		),
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.AlertEvent{}, false, nil
		}

		return domain.AlertEvent{}, false, err
	}

	return alert, true, nil
}

// Resolve marks an alert event as resolved.
func (r *AlertRepository) Resolve(
	ctx context.Context,
	id domain.AlertID,
) (domain.AlertEvent, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	alert, err := scanAlertRow(
		r.pool.QueryRow(
			ctx,
			`
			UPDATE alert_events
			SET
				status = $2,
				resolved_at = now()
			WHERE id = $1
			RETURNING
				id,
				parameter_type,
				level,
				status,
				value,
				unit,
				source_id,
				message,
				created_at,
				acknowledged_at,
				acknowledged_by,
				resolved_at
			`,
			id,
			domain.AlertStatusResolved,
		),
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.AlertEvent{}, false, nil
		}

		return domain.AlertEvent{}, false, err
	}

	return alert, true, nil
}

type alertRowScanner interface {
	Scan(dest ...any) error
}

func scanAlerts(rows pgx.Rows) ([]domain.AlertEvent, error) {
	alerts := make([]domain.AlertEvent, 0)

	for rows.Next() {
		alert, err := scanAlertRow(rows)
		if err != nil {
			return nil, err
		}

		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return alerts, nil
}

func scanAlertRow(row alertRowScanner) (domain.AlertEvent, error) {
	var alert domain.AlertEvent
	var acknowledgedAt pgtype.Timestamptz
	var acknowledgedBy pgtype.Int8
	var resolvedAt pgtype.Timestamptz

	err := row.Scan(
		&alert.ID,
		&alert.ParameterType,
		&alert.Level,
		&alert.Status,
		&alert.Value,
		&alert.Unit,
		&alert.SourceID,
		&alert.Message,
		&alert.CreatedAt,
		&acknowledgedAt,
		&acknowledgedBy,
		&resolvedAt,
	)
	if err != nil {
		return domain.AlertEvent{}, err
	}

	if acknowledgedAt.Valid {
		value := acknowledgedAt.Time
		alert.AcknowledgedAt = &value
	}

	if acknowledgedBy.Valid {
		value := domain.UserID(acknowledgedBy.Int64)
		alert.AcknowledgedBy = &value
	}

	if resolvedAt.Valid {
		value := resolvedAt.Time
		alert.ResolvedAt = &value
	}

	return alert, nil
}
