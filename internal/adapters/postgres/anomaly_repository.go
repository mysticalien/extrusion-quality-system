package postgres

import (
	"context"
	"time"

	"extrusion-quality-system/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnomalyRepository struct {
	pool *pgxpool.Pool
}

func NewAnomalyRepository(pool *pgxpool.Pool) *AnomalyRepository {
	return &AnomalyRepository{pool: pool}
}

func (r *AnomalyRepository) Create(
	ctx context.Context,
	anomaly domain.AnomalyEvent,
) (domain.AnomalyEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	now := time.Now().UTC()

	if anomaly.Status == "" {
		anomaly.Status = domain.AlertStatusActive
	}

	if anomaly.CreatedAt.IsZero() {
		anomaly.CreatedAt = now
	}

	if anomaly.UpdatedAt.IsZero() {
		anomaly.UpdatedAt = now
	}

	if anomaly.ObservedAt.IsZero() {
		anomaly.ObservedAt = now
	}

	err := r.pool.QueryRow(
		ctx,
		`
		INSERT INTO anomaly_events (
			type,
			parameter_type,
			level,
			status,
			message,
			current_value,
			previous_value,
			source_id,
			observed_at,
			created_at,
			updated_at,
			resolved_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
		`,
		anomaly.Type,
		anomaly.ParameterType,
		anomaly.Level,
		anomaly.Status,
		anomaly.Message,
		anomaly.CurrentValue,
		anomaly.PreviousValue,
		anomaly.SourceID,
		anomaly.ObservedAt,
		anomaly.CreatedAt,
		anomaly.UpdatedAt,
		anomaly.ResolvedAt,
	).Scan(&anomaly.ID)

	if err != nil {
		return domain.AnomalyEvent{}, err
	}

	return anomaly, nil
}

func (r *AnomalyRepository) All(ctx context.Context) ([]domain.AnomalyEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := r.pool.Query(
		ctx,
		`
		SELECT
			id,
			type,
			parameter_type,
			level,
			status,
			message,
			current_value,
			previous_value,
			source_id,
			observed_at,
			created_at,
			updated_at,
			resolved_at
		FROM anomaly_events
		ORDER BY created_at DESC, id DESC
		`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAnomalies(rows)
}

func (r *AnomalyRepository) Active(ctx context.Context) ([]domain.AnomalyEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := r.pool.Query(
		ctx,
		`
		SELECT
			id,
			type,
			parameter_type,
			level,
			status,
			message,
			current_value,
			previous_value,
			source_id,
			observed_at,
			created_at,
			updated_at,
			resolved_at
		FROM (
			SELECT DISTINCT ON (type, parameter_type)
				id,
				type,
				parameter_type,
				level,
				status,
				message,
				current_value,
				previous_value,
				source_id,
				observed_at,
				created_at,
				updated_at,
				resolved_at
			FROM anomaly_events
			WHERE status IN ($1, $2)
			ORDER BY type, parameter_type, updated_at DESC, id DESC
		) latest_open_anomalies
		ORDER BY updated_at DESC, id DESC
		`,
		domain.AlertStatusActive,
		domain.AlertStatusAcknowledged,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAnomalies(rows)
}

func (r *AnomalyRepository) FindOpenByTypeAndParameter(
	ctx context.Context,
	anomalyType domain.AnomalyType,
	parameterType domain.ParameterType,
) (domain.AnomalyEvent, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	anomaly, err := scanAnomalyRow(
		r.pool.QueryRow(
			ctx,
			`
			SELECT
				id,
				type,
				parameter_type,
				level,
				status,
				message,
				current_value,
				previous_value,
				source_id,
				observed_at,
				created_at,
				updated_at,
				resolved_at
			FROM anomaly_events
			WHERE type = $1
			  AND parameter_type = $2
			  AND status IN ($3, $4)
			ORDER BY updated_at DESC, id DESC
			LIMIT 1
			`,
			anomalyType,
			parameterType,
			domain.AlertStatusActive,
			domain.AlertStatusAcknowledged,
		),
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.AnomalyEvent{}, false, nil
		}

		return domain.AnomalyEvent{}, false, err
	}

	return anomaly, true, nil
}

func (r *AnomalyRepository) UpdateOpen(
	ctx context.Context,
	anomaly domain.AnomalyEvent,
) (domain.AnomalyEvent, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	updated, err := scanAnomalyRow(
		r.pool.QueryRow(
			ctx,
			`
			UPDATE anomaly_events
			SET
				level = $2,
				message = $3,
				current_value = $4,
				previous_value = $5,
				source_id = $6,
				observed_at = $7,
				updated_at = now()
			WHERE id = $1
			  AND status IN ($8, $9)
			RETURNING
				id,
				type,
				parameter_type,
				level,
				status,
				message,
				current_value,
				previous_value,
				source_id,
				observed_at,
				created_at,
				updated_at,
				resolved_at
			`,
			anomaly.ID,
			anomaly.Level,
			anomaly.Message,
			anomaly.CurrentValue,
			anomaly.PreviousValue,
			anomaly.SourceID,
			anomaly.ObservedAt,
			domain.AlertStatusActive,
			domain.AlertStatusAcknowledged,
		),
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.AnomalyEvent{}, false, nil
		}

		return domain.AnomalyEvent{}, false, err
	}

	return updated, true, nil
}

func (r *AnomalyRepository) ResolveOpenByTypeAndParameter(
	ctx context.Context,
	anomalyType domain.AnomalyType,
	parameterType domain.ParameterType,
) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	commandTag, err := r.pool.Exec(
		ctx,
		`
		UPDATE anomaly_events
		SET
			status = $3,
			resolved_at = now(),
			updated_at = now()
		WHERE type = $1
		  AND parameter_type = $2
		  AND status IN ($4, $5)
		`,
		anomalyType,
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

type anomalyRowScanner interface {
	Scan(dest ...any) error
}

func scanAnomalies(rows pgx.Rows) ([]domain.AnomalyEvent, error) {
	anomalies := make([]domain.AnomalyEvent, 0)

	for rows.Next() {
		anomaly, err := scanAnomalyRow(rows)
		if err != nil {
			return nil, err
		}

		anomalies = append(anomalies, anomaly)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return anomalies, nil
}

func scanAnomalyRow(row anomalyRowScanner) (domain.AnomalyEvent, error) {
	var anomaly domain.AnomalyEvent
	var currentValue pgtype.Float8
	var previousValue pgtype.Float8
	var resolvedAt pgtype.Timestamptz

	err := row.Scan(
		&anomaly.ID,
		&anomaly.Type,
		&anomaly.ParameterType,
		&anomaly.Level,
		&anomaly.Status,
		&anomaly.Message,
		&currentValue,
		&previousValue,
		&anomaly.SourceID,
		&anomaly.ObservedAt,
		&anomaly.CreatedAt,
		&anomaly.UpdatedAt,
		&resolvedAt,
	)
	if err != nil {
		return domain.AnomalyEvent{}, err
	}

	if currentValue.Valid {
		value := currentValue.Float64
		anomaly.CurrentValue = &value
	}

	if previousValue.Valid {
		value := previousValue.Float64
		anomaly.PreviousValue = &value
	}

	if resolvedAt.Valid {
		value := resolvedAt.Time
		anomaly.ResolvedAt = &value
	}

	return anomaly, nil
}
