CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_unique
    ON users (username);