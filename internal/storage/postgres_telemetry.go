package storage

import (
	"context"
	"time"

	"extrusion-quality-system/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresTelemetryStore stores telemetry readings in PostgreSQL.
type PostgresTelemetryStore struct {
	pool *pgxpool.Pool
}

// NewPostgresTelemetryStore creates a PostgreSQL telemetry store.
func NewPostgresTelemetryStore(pool *pgxpool.Pool) *PostgresTelemetryStore {
	return &PostgresTelemetryStore{
		pool: pool,
	}
}

// Save stores a telemetry reading in PostgreSQL.
func (s *PostgresTelemetryStore) Save(reading domain.TelemetryReading) (domain.TelemetryReading, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if reading.CreatedAt.IsZero() {
		reading.CreatedAt = time.Now().UTC()
	}

	err := s.pool.QueryRow(
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
func (s *PostgresTelemetryStore) All() ([]domain.TelemetryReading, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := s.pool.Query(
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
