-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS process_parameters (
                                                  code text PRIMARY KEY,
                                                  name text NOT NULL,
                                                  unit text NOT NULL,
                                                  description text NOT NULL DEFAULT ''
);

INSERT INTO process_parameters (code, name, unit, description) VALUES
                                                                   ('pressure', 'Давление', 'bar', 'Давление в рабочей зоне экструдера'),
                                                                   ('moisture', 'Влажность', 'percent', 'Влажность сырья или смеси'),
                                                                   ('barrel_temperature_zone_1', 'Температура зоны 1', 'celsius', 'Температура первой зоны цилиндра экструдера'),
                                                                   ('barrel_temperature_zone_2', 'Температура зоны 2', 'celsius', 'Температура второй зоны цилиндра экструдера'),
                                                                   ('barrel_temperature_zone_3', 'Температура зоны 3', 'celsius', 'Температура третьей зоны цилиндра экструдера'),
                                                                   ('screw_speed', 'Скорость шнека', 'rpm', 'Частота вращения шнека экструдера'),
                                                                   ('drive_load', 'Нагрузка привода', 'percent', 'Текущая нагрузка электропривода экструдера'),
                                                                   ('outlet_temperature', 'Температура на выходе', 'celsius', 'Температура продукта на выходе из экструдера'),
                                                                   ('process_risk', 'Риск нестабильности процесса', 'state', 'Расчётный параметр для комбинированных аномалий процесса')
ON CONFLICT (code) DO UPDATE SET
                                 name = EXCLUDED.name,
                                 unit = EXCLUDED.unit,
                                 description = EXCLUDED.description;

-- На случай, если в существующих данных уже есть параметры,
-- которых нет в базовом справочнике.
INSERT INTO process_parameters (code, name, unit, description)
SELECT DISTINCT
    telemetry_readings.parameter_type,
    telemetry_readings.parameter_type,
    telemetry_readings.unit,
    'Автоматически добавленный параметр из существующей телеметрии'
FROM telemetry_readings
         LEFT JOIN process_parameters
                   ON process_parameters.code = telemetry_readings.parameter_type
WHERE process_parameters.code IS NULL
  AND telemetry_readings.parameter_type IS NOT NULL
ON CONFLICT (code) DO NOTHING;

INSERT INTO process_parameters (code, name, unit, description)
SELECT DISTINCT
    alert_events.parameter_type,
    alert_events.parameter_type,
    alert_events.unit,
    'Автоматически добавленный параметр из существующих событий'
FROM alert_events
         LEFT JOIN process_parameters
                   ON process_parameters.code = alert_events.parameter_type
WHERE process_parameters.code IS NULL
  AND alert_events.parameter_type IS NOT NULL
ON CONFLICT (code) DO NOTHING;

INSERT INTO process_parameters (code, name, unit, description)
SELECT DISTINCT
    anomaly_events.parameter_type,
    anomaly_events.parameter_type,
    'state',
    'Автоматически добавленный параметр из существующих аномалий'
FROM anomaly_events
         LEFT JOIN process_parameters
                   ON process_parameters.code = anomaly_events.parameter_type
WHERE process_parameters.code IS NULL
  AND anomaly_events.parameter_type IS NOT NULL
ON CONFLICT (code) DO NOTHING;

INSERT INTO process_parameters (code, name, unit, description)
SELECT DISTINCT
    quality_weights.parameter,
    quality_weights.parameter,
    'weight',
    'Автоматически добавленный параметр из весов индекса качества'
FROM quality_weights
         LEFT JOIN process_parameters
                   ON process_parameters.code = quality_weights.parameter
WHERE process_parameters.code IS NULL
  AND quality_weights.parameter IS NOT NULL
ON CONFLICT (code) DO NOTHING;

-- Чистим некорректные ссылки на пользователей перед добавлением FK.
UPDATE setpoints
SET updated_by = NULL
WHERE updated_by IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM users
    WHERE users.id = setpoints.updated_by
);

UPDATE alert_events
SET acknowledged_by = NULL
WHERE acknowledged_by IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM users
    WHERE users.id = alert_events.acknowledged_by
);

-- Добавляем FK безопасно: если constraint уже есть, миграция не упадёт.
DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'fk_telemetry_readings_parameter'
        ) THEN
            ALTER TABLE telemetry_readings
                ADD CONSTRAINT fk_telemetry_readings_parameter
                    FOREIGN KEY (parameter_type)
                        REFERENCES process_parameters(code)
                        ON UPDATE CASCADE
                        ON DELETE RESTRICT;
        END IF;
    END $$;

DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'fk_setpoints_parameter'
        ) THEN
            ALTER TABLE setpoints
                ADD CONSTRAINT fk_setpoints_parameter
                    FOREIGN KEY (parameter_type)
                        REFERENCES process_parameters(code)
                        ON UPDATE CASCADE
                        ON DELETE RESTRICT;
        END IF;
    END $$;

DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'fk_alert_events_parameter'
        ) THEN
            ALTER TABLE alert_events
                ADD CONSTRAINT fk_alert_events_parameter
                    FOREIGN KEY (parameter_type)
                        REFERENCES process_parameters(code)
                        ON UPDATE CASCADE
                        ON DELETE RESTRICT;
        END IF;
    END $$;

DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'fk_anomaly_events_parameter'
        ) THEN
            ALTER TABLE anomaly_events
                ADD CONSTRAINT fk_anomaly_events_parameter
                    FOREIGN KEY (parameter_type)
                        REFERENCES process_parameters(code)
                        ON UPDATE CASCADE
                        ON DELETE RESTRICT;
        END IF;
    END $$;

DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'fk_quality_weights_parameter'
        ) THEN
            ALTER TABLE quality_weights
                ADD CONSTRAINT fk_quality_weights_parameter
                    FOREIGN KEY (parameter)
                        REFERENCES process_parameters(code)
                        ON UPDATE CASCADE
                        ON DELETE RESTRICT;
        END IF;
    END $$;

DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'fk_setpoints_updated_by'
        ) THEN
            ALTER TABLE setpoints
                ADD CONSTRAINT fk_setpoints_updated_by
                    FOREIGN KEY (updated_by)
                        REFERENCES users(id)
                        ON UPDATE CASCADE
                        ON DELETE SET NULL;
        END IF;
    END $$;

DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'fk_alert_events_acknowledged_by'
        ) THEN
            ALTER TABLE alert_events
                ADD CONSTRAINT fk_alert_events_acknowledged_by
                    FOREIGN KEY (acknowledged_by)
                        REFERENCES users(id)
                        ON UPDATE CASCADE
                        ON DELETE SET NULL;
        END IF;
    END $$;

CREATE INDEX IF NOT EXISTS idx_process_parameters_unit
    ON process_parameters (unit);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE alert_events
    DROP CONSTRAINT IF EXISTS fk_alert_events_acknowledged_by;

ALTER TABLE setpoints
    DROP CONSTRAINT IF EXISTS fk_setpoints_updated_by;

ALTER TABLE quality_weights
    DROP CONSTRAINT IF EXISTS fk_quality_weights_parameter;

ALTER TABLE anomaly_events
    DROP CONSTRAINT IF EXISTS fk_anomaly_events_parameter;

ALTER TABLE alert_events
    DROP CONSTRAINT IF EXISTS fk_alert_events_parameter;

ALTER TABLE setpoints
    DROP CONSTRAINT IF EXISTS fk_setpoints_parameter;

ALTER TABLE telemetry_readings
    DROP CONSTRAINT IF EXISTS fk_telemetry_readings_parameter;

DROP INDEX IF EXISTS idx_process_parameters_unit;

DROP TABLE IF EXISTS process_parameters;

-- +goose StatementEnd