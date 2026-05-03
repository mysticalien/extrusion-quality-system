package postgres

import (
	"context"
	"time"

	"extrusion-quality-system/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QualityRepository stores quality index values in PostgreSQL.
type QualityRepository struct {
	pool *pgxpool.Pool
}

// NewQualityRepository creates a PostgreSQL quality repository.
func NewQualityRepository(pool *pgxpool.Pool) *QualityRepository {
	return &QualityRepository{
		pool: pool,
	}
}

// Save stores a quality index value in PostgreSQL.
func (r *QualityRepository) Save(
	ctx context.Context,
	index domain.QualityIndex,
) (domain.QualityIndex, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := r.pool.QueryRow(
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
func (r *QualityRepository) Latest(ctx context.Context) (domain.QualityIndex, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var index domain.QualityIndex

	err := r.pool.QueryRow(
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

// History returns quality index values in the given time range.
// It selects the newest records first by limit, then returns them
// in chronological order for charts.
func (r *QualityRepository) History(
	ctx context.Context,
	from time.Time,
	to time.Time,
	limit int,
) ([]domain.QualityIndex, error) {
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
			value,
			state,
			parameter_penalty,
			anomaly_penalty,
			calculated_at
		FROM (
			SELECT
				id,
				value,
				state,
				parameter_penalty,
				anomaly_penalty,
				calculated_at
			FROM quality_index_values
			WHERE ($1::timestamptz IS NULL OR calculated_at >= $1)
			  AND ($2::timestamptz IS NULL OR calculated_at <= $2)
			ORDER BY calculated_at DESC, id DESC
			LIMIT $3
		) latest_values
		ORDER BY calculated_at ASC, id ASC
		`,
		nullableTime(from),
		nullableTime(to),
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.QualityIndex, 0)

	for rows.Next() {
		var index domain.QualityIndex

		if err := rows.Scan(
			&index.ID,
			&index.Value,
			&index.State,
			&index.ParameterPenalty,
			&index.AnomalyPenalty,
			&index.CalculatedAt,
		); err != nil {
			return nil, err
		}

		result = append(result, index)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
