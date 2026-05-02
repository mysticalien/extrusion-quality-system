package storage

import (
	"context"
	"time"

	"extrusion-quality-system/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresSetpointRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSetpointRepository(pool *pgxpool.Pool) *PostgresSetpointRepository {
	return &PostgresSetpointRepository{
		pool: pool,
	}
}

func (r *PostgresSetpointRepository) All(ctx context.Context) ([]domain.Setpoint, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := r.pool.Query(
		ctx,
		`
		SELECT
			id,
			parameter_type,
			unit,
			critical_min,
			warning_min,
			normal_min,
			normal_max,
			warning_max,
			critical_max,
			created_at,
			updated_at,
			updated_by
		FROM setpoints
		ORDER BY parameter_type
		`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	setpoints := make([]domain.Setpoint, 0)

	for rows.Next() {
		setpoint, err := scanSetpointRow(rows)
		if err != nil {
			return nil, err
		}

		setpoints = append(setpoints, setpoint)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return setpoints, nil
}

func (r *PostgresSetpointRepository) GetByParameter(
	ctx context.Context,
	parameterType domain.ParameterType,
) (domain.Setpoint, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	setpoint, err := scanSetpointRow(
		r.pool.QueryRow(
			ctx,
			`
			SELECT
				id,
				parameter_type,
				unit,
				critical_min,
				warning_min,
				normal_min,
				normal_max,
				warning_max,
				critical_max,
				created_at,
				updated_at,
				updated_by
			FROM setpoints
			WHERE parameter_type = $1
			`,
			parameterType,
		),
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Setpoint{}, false, nil
		}

		return domain.Setpoint{}, false, err
	}

	return setpoint, true, nil
}

type setpointRowScanner interface {
	Scan(dest ...any) error
}

func scanSetpointRow(row setpointRowScanner) (domain.Setpoint, error) {
	var setpoint domain.Setpoint
	var updatedBy pgtype.Int8

	err := row.Scan(
		&setpoint.ID,
		&setpoint.ParameterType,
		&setpoint.Unit,
		&setpoint.CriticalMin,
		&setpoint.WarningMin,
		&setpoint.NormalMin,
		&setpoint.NormalMax,
		&setpoint.WarningMax,
		&setpoint.CriticalMax,
		&setpoint.CreatedAt,
		&setpoint.UpdatedAt,
		&updatedBy,
	)
	if err != nil {
		return domain.Setpoint{}, err
	}

	if updatedBy.Valid {
		value := domain.UserID(updatedBy.Int64)
		setpoint.UpdatedBy = &value
	}

	return setpoint, nil
}
