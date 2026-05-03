package postgres

import (
	"context"
	"errors"
	"time"

	"extrusion-quality-system/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SetpointRepository struct {
	pool *pgxpool.Pool
}

func NewSetpointRepository(pool *pgxpool.Pool) *SetpointRepository {
	return &SetpointRepository{
		pool: pool,
	}
}

func (r *SetpointRepository) All(ctx context.Context) ([]domain.Setpoint, error) {
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

func (r *SetpointRepository) GetByParameter(
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

func (r *SetpointRepository) Update(
	ctx context.Context,
	id int64,
	update domain.SetpointUpdate,
) (domain.Setpoint, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	const query = `
		UPDATE setpoints
		SET
			critical_min = $2,
			warning_min = $3,
			normal_min = $4,
			normal_max = $5,
			warning_max = $6,
			critical_max = $7,
			updated_at = now()
		WHERE id = $1
		RETURNING
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
			updated_at
	`

	var setpoint domain.Setpoint

	err := r.pool.QueryRow(
		ctx,
		query,
		id,
		update.CriticalMin,
		update.WarningMin,
		update.NormalMin,
		update.NormalMax,
		update.WarningMax,
		update.CriticalMax,
	).Scan(
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
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Setpoint{}, false, nil
	}

	if err != nil {
		return domain.Setpoint{}, false, err
	}

	return setpoint, true, nil
}
