CREATE TABLE IF NOT EXISTS users (
                                     id BIGSERIAL PRIMARY KEY,
                                     username TEXT NOT NULL UNIQUE,
                                     password_hash TEXT NOT NULL,
                                     role TEXT NOT NULL,
                                     is_active BOOLEAN NOT NULL DEFAULT true,
                                     created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                     updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

                                     CONSTRAINT users_role_valid CHECK (role IN ('operator', 'technologist', 'admin'))
);

CREATE INDEX IF NOT EXISTS idx_users_username
    ON users (username);

INSERT INTO users (
    username,
    password_hash,
    role,
    is_active
)
VALUES
    (
        'operator',
        '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
        'operator',
        true
    ),
    (
        'technologist',
        '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
        'technologist',
        true
    ),
    (
        'admin',
        '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
        'admin',
        true
    )
ON CONFLICT (username) DO UPDATE
    SET
        password_hash = EXCLUDED.password_hash,
        role = EXCLUDED.role,
        is_active = EXCLUDED.is_active,
        updated_at = now();