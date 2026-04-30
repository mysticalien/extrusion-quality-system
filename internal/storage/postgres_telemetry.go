package storage

import (
	"context"
	"time"

	"extrusion-quality-system/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresTelemetryRepository stores telemetry readings in PostgreSQL.
type PostgresTelemetryRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresTelemetryRepository creates a PostgreSQL telemetry repository.
func NewPostgresTelemetryRepository(pool *pgxpool.Pool) *PostgresTelemetryRepository {
	return &PostgresTelemetryRepository{
		pool: pool,
	}
}

// Save stores a telemetry reading in PostgreSQL.
func (r *PostgresTelemetryRepository) Save(
	ctx context.Context,
	reading domain.TelemetryReading,
) (domain.TelemetryReading, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if reading.CreatedAt.IsZero() {
		reading.CreatedAt = time.Now().UTC()
	}

	err := r.pool.QueryRow(
		ctx,
		`
		INSERT INTO telemetry_readings (
			parameter_type,
			value,
			unit,
			source_id,
			measured_at,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
		`,
		reading.ParameterType,
		reading.Value,
		reading.Unit,
		reading.SourceID,
		reading.MeasuredAt,
		reading.CreatedAt,
	).Scan(&reading.ID)

	if err != nil {
		return domain.TelemetryReading{}, err
	}

	return reading, nil
}

// All returns all telemetry readings ordered by ID.
func (r *PostgresTelemetryRepository) All(ctx context.Context) ([]domain.TelemetryReading, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := r.pool.Query(
		ctx,
		`
		SELECT
			id,
			parameter_type,
			value,
			unit,
			source_id,
			measured_at,
			created_at
		FROM telemetry_readings
		ORDER BY id
		`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTelemetryReadings(rows)
}

// Latest returns the latest reading for each parameter type.
func (r *PostgresTelemetryRepository) Latest(ctx context.Context) ([]domain.TelemetryReading, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := r.pool.Query(
		ctx,
		`
		SELECT DISTINCT ON (parameter_type)
			id,
			parameter_type,
			value,
			unit,
			source_id,
			measured_at,
			created_at
		FROM telemetry_readings
		ORDER BY parameter_type, measured_at DESC, id DESC
		`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTelemetryReadings(rows)
}

// HistoryByParameter returns telemetry readings for one parameter in the given time range.
func (r *PostgresTelemetryRepository) HistoryByParameter(
	ctx context.Context,
	parameterType domain.ParameterType,
	from time.Time,
	to time.Time,
	limit int,
) ([]domain.TelemetryReading, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if limit <= 0 {
		limit = 100
	}

	rows, err := r.pool.Query(
		ctx,
		`
		SELECT
			id,
			parameter_type,
			value,
			unit,
			source_id,
			measured_at,
			created_at
		FROM telemetry_readings
		WHERE parameter_type = $1
		  AND ($2::timestamptz IS NULL OR measured_at >= $2)
		  AND ($3::timestamptz IS NULL OR measured_at <= $3)
		ORDER BY measured_at ASC, id ASC
		LIMIT $4
		`,
		parameterType,
		nullableTime(from),
		nullableTime(to),
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTelemetryReadings(rows)
}

type telemetryRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanTelemetryReadings(rows telemetryRows) ([]domain.TelemetryReading, error) {
	readings := make([]domain.TelemetryReading, 0)

	for rows.Next() {
		var reading domain.TelemetryReading

		if err := rows.Scan(
			&reading.ID,
			&reading.ParameterType,
			&reading.Value,
			&reading.Unit,
			&reading.SourceID,
			&reading.MeasuredAt,
			&reading.CreatedAt,
		); err != nil {
			return nil, err
		}

		readings = append(readings, reading)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return readings, nil
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}

	return value
}
