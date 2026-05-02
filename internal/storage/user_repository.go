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

func (r *PostgresUserRepository) All(ctx context.Context) ([]domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := r.pool.Query(
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
		ORDER BY id
		`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]domain.User, 0)

	for rows.Next() {
		user, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
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

func (r *PostgresUserRepository) Create(
	ctx context.Context,
	user domain.User,
) (domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}

	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = user.CreatedAt
	}

	created, err := scanUserRow(
		r.pool.QueryRow(
			ctx,
			`
			INSERT INTO users (
				username,
				password_hash,
				role,
				is_active,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING
				id,
				username,
				password_hash,
				role,
				is_active,
				created_at,
				updated_at
			`,
			user.Username,
			user.PasswordHash,
			user.Role,
			user.IsActive,
			user.CreatedAt,
			user.UpdatedAt,
		),
	)

	if err != nil {
		return domain.User{}, err
	}

	return created, nil
}

func (r *PostgresUserRepository) UpdateRole(
	ctx context.Context,
	id domain.UserID,
	role domain.UserRole,
) (domain.User, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	user, err := scanUserRow(
		r.pool.QueryRow(
			ctx,
			`
			UPDATE users
			SET
				role = $2,
				updated_at = now()
			WHERE id = $1
			RETURNING
				id,
				username,
				password_hash,
				role,
				is_active,
				created_at,
				updated_at
			`,
			id,
			role,
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

func (r *PostgresUserRepository) UpdatePassword(
	ctx context.Context,
	id domain.UserID,
	passwordHash string,
) (domain.User, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	user, err := scanUserRow(
		r.pool.QueryRow(
			ctx,
			`
			UPDATE users
			SET
				password_hash = $2,
				updated_at = now()
			WHERE id = $1
			RETURNING
				id,
				username,
				password_hash,
				role,
				is_active,
				created_at,
				updated_at
			`,
			id,
			passwordHash,
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

func (r *PostgresUserRepository) SetActive(
	ctx context.Context,
	id domain.UserID,
	isActive bool,
) (domain.User, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	user, err := scanUserRow(
		r.pool.QueryRow(
			ctx,
			`
			UPDATE users
			SET
				is_active = $2,
				updated_at = now()
			WHERE id = $1
			RETURNING
				id,
				username,
				password_hash,
				role,
				is_active,
				created_at,
				updated_at
			`,
			id,
			isActive,
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
