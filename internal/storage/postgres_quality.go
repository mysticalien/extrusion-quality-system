package storage

import (
	"context"
	"time"

	"extrusion-quality-system/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresQualityStore stores quality index values in PostgreSQL.
type PostgresQualityStore struct {
	pool *pgxpool.Pool
}

// NewPostgresQualityStore creates a PostgreSQL quality index store.
func NewPostgresQualityStore(pool *pgxpool.Pool) *PostgresQualityStore {
	return &PostgresQualityStore{
		pool: pool,
	}
}

// Save stores a quality index value in PostgreSQL.
func (s *PostgresQualityStore) Save(index domain.QualityIndex) (domain.QualityIndex, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := s.pool.QueryRow(
		ctx,
		`
		INSERT INTO quality_index_values (
			value,
			state,
			parameter_penalty,
			anomaly_penalty,
			calculated_at
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
		`,
		index.Value,
		index.State,
		index.ParameterPenalty,
		index.AnomalyPenalty,
		index.CalculatedAt,
	).Scan(&index.ID)

	if err != nil {
		return domain.QualityIndex{}, err
	}

	return index, nil
}

// Latest returns the latest stored quality index value.
func (s *PostgresQualityStore) Latest() (domain.QualityIndex, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var index domain.QualityIndex

	err := s.pool.QueryRow(
		ctx,
		`
		SELECT
			id,
			value,
			state,
			parameter_penalty,
			anomaly_penalty,
			calculated_at
		FROM quality_index_values
		ORDER BY calculated_at DESC, id DESC
		LIMIT 1
		`,
	).Scan(
		&index.ID,
		&index.Value,
		&index.State,
		&index.ParameterPenalty,
		&index.AnomalyPenalty,
		&index.CalculatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.QualityIndex{}, false, nil
		}

		return domain.QualityIndex{}, false, err
	}

	return index, true, nil
}
