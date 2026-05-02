-- +goose Up
-- +goose StatementBegin

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

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS users;

-- +goose StatementEnd