package storage

import (
	"context"
	"time"

	"extrusion-quality-system/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{
		pool: pool,
	}
}

func (r *PostgresUserRepository) FindByUsername(
	ctx context.Context,
	username string,
) (domain.User, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	user, err := scanUserRow(
		r.pool.QueryRow(
			ctx,
			`
			SELECT
				id,
				username,
				password_hash,
				role,
				is_active,
				created_at,
				updated_at
			FROM users
			WHERE username = $1
			`,
			username,
		),
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.User{}, false, nil
		}

		return domain.User{}, false, err
	}

	return user, true, nil
}

func (r *PostgresUserRepository) FindByID(
	ctx context.Context,
	id domain.UserID,
) (domain.User, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	user, err := scanUserRow(
		r.pool.QueryRow(
			ctx,
			`
			SELECT
				id,
				username,
				password_hash,
				role,
				is_active,
				created_at,
				updated_at
			FROM users
			WHERE id = $1
			`,
			id,
		),
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.User{}, false, nil
		}

		return domain.User{}, false, err
	}

	return user, true, nil
}

type userRowScanner interface {
	Scan(dest ...any) error
}

func scanUserRow(row userRowScanner) (domain.User, error) {
	var user domain.User

	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}
