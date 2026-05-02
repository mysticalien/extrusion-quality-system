-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS quality_weights (
                                               id BIGSERIAL PRIMARY KEY,
                                               parameter TEXT NOT NULL UNIQUE,
                                               weight NUMERIC(8, 3) NOT NULL CHECK (weight > 0),
                                               created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                               updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                               updated_by TEXT
);

INSERT INTO quality_weights (
    parameter,
    weight,
    created_at,
    updated_at,
    updated_by
)
VALUES
    ('pressure', 2.000, now(), now(), 'migration'),
    ('moisture', 1.500, now(), now(), 'migration'),
    ('drive_load', 1.700, now(), now(), 'migration'),
    ('barrel_temperature_zone_1', 1.000, now(), now(), 'migration'),
    ('barrel_temperature_zone_2', 1.200, now(), now(), 'migration'),
    ('barrel_temperature_zone_3', 1.400, now(), now(), 'migration'),
    ('screw_speed', 0.600, now(), now(), 'migration'),
    ('outlet_temperature', 0.800, now(), now(), 'migration')
ON CONFLICT (parameter)
    DO UPDATE SET
                  weight = EXCLUDED.weight,
                  updated_at = now(),
                  updated_by = 'migration';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS quality_weights;

-- +goose StatementEnd