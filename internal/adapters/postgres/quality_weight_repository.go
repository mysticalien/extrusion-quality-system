package postgres

import (
	"context"
	"fmt"
	"time"

	"extrusion-quality-system/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QualityWeightRepository struct {
	pool *pgxpool.Pool
}

func NewQualityWeightRepository(pool *pgxpool.Pool) *QualityWeightRepository {
	return &QualityWeightRepository{
		pool: pool,
	}
}

func (r *QualityWeightRepository) List(ctx context.Context) ([]domain.QualityWeight, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := r.pool.Query(ctx, `
		SELECT
			id,
			parameter,
			weight,
			created_at,
			updated_at,
			COALESCE(updated_by, '')
		FROM quality_weights
		ORDER BY parameter
	`)
	if err != nil {
		return nil, fmt.Errorf("query quality weights: %w", err)
	}
	defer rows.Close()

	var weights []domain.QualityWeight

	for rows.Next() {
		var weight domain.QualityWeight
		var rawParameter string

		if err := rows.Scan(
			&weight.ID,
			&rawParameter,
			&weight.Weight,
			&weight.CreatedAt,
			&weight.UpdatedAt,
			&weight.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan quality weight: %w", err)
		}

		weight.ParameterType = domain.ParameterType(rawParameter)

		weights = append(weights, weight)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate quality weights: %w", err)
	}

	return weights, nil
}

func (r *QualityWeightRepository) Update(
	ctx context.Context,
	id domain.QualityWeightID,
	update domain.QualityWeightUpdate,
	updatedBy string,
) (domain.QualityWeight, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var weight domain.QualityWeight
	var rawParameter string

	err := r.pool.QueryRow(ctx, `
		UPDATE quality_weights
		SET
			weight = $2,
			updated_at = now(),
			updated_by = $3
		WHERE id = $1
		RETURNING
			id,
			parameter,
			weight,
			created_at,
			updated_at,
			COALESCE(updated_by, '')
	`, id, update.Weight, updatedBy).Scan(
		&weight.ID,
		&rawParameter,
		&weight.Weight,
		&weight.CreatedAt,
		&weight.UpdatedAt,
		&weight.UpdatedBy,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.QualityWeight{}, false, nil
		}

		return domain.QualityWeight{}, false, fmt.Errorf("update quality weight: %w", err)
	}

	weight.ParameterType = domain.ParameterType(rawParameter)

	return weight, true, nil
}
