CREATE TABLE IF NOT EXISTS setpoints (
                                         id BIGSERIAL PRIMARY KEY,
                                         parameter_type TEXT NOT NULL UNIQUE,
                                         unit TEXT NOT NULL,

                                         critical_min DOUBLE PRECISION NOT NULL,
                                         warning_min DOUBLE PRECISION NOT NULL,
                                         normal_min DOUBLE PRECISION NOT NULL,
                                         normal_max DOUBLE PRECISION NOT NULL,
                                         warning_max DOUBLE PRECISION NOT NULL,
                                         critical_max DOUBLE PRECISION NOT NULL,

                                         created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                         updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                         updated_by BIGINT,

                                         CONSTRAINT setpoints_ranges_valid CHECK (
                                             critical_min <= warning_min
                                                 AND warning_min <= normal_min
                                                 AND normal_min <= normal_max
                                                 AND normal_max <= warning_max
                                                 AND warning_max <= critical_max
                                             )
);

INSERT INTO setpoints (
    parameter_type,
    unit,
    critical_min,
    warning_min,
    normal_min,
    normal_max,
    warning_max,
    critical_max
)
VALUES
    ('pressure', 'bar', 30, 35, 40, 75, 90, 95),
    ('moisture', 'percent', 15, 20, 22, 28, 30, 35),
    ('barrel_temperature_zone_1', 'celsius', 70, 80, 90, 120, 130, 140),
    ('barrel_temperature_zone_2', 'celsius', 80, 90, 100, 140, 150, 160),
    ('barrel_temperature_zone_3', 'celsius', 90, 100, 110, 150, 160, 170),
    ('screw_speed', 'rpm', 100, 150, 200, 450, 500, 550),
    ('drive_load', 'percent', 20, 30, 40, 80, 90, 100),
    ('outlet_temperature', 'celsius', 70, 80, 90, 130, 140, 150)
ON CONFLICT (parameter_type) DO UPDATE
    SET
        unit = EXCLUDED.unit,
        critical_min = EXCLUDED.critical_min,
        warning_min = EXCLUDED.warning_min,
        normal_min = EXCLUDED.normal_min,
        normal_max = EXCLUDED.normal_max,
        warning_max = EXCLUDED.warning_max,
        critical_max = EXCLUDED.critical_max,
        updated_at = now();

CREATE INDEX IF NOT EXISTS idx_setpoints_parameter_type
    ON setpoints (parameter_type);