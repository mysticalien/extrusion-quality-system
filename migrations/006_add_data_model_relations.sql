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

-- Добавляем в справочник все параметры, которые уже могли появиться в телеметрии.
INSERT INTO process_parameters (code, name, unit, description)
SELECT DISTINCT
    tr.parameter_type,
    tr.parameter_type,
    tr.unit,
    'Автоматически добавленный параметр из существующих телеметрических данных'
FROM telemetry_readings tr
         LEFT JOIN process_parameters pp ON pp.code = tr.parameter_type
WHERE pp.code IS NULL
  AND tr.parameter_type IS NOT NULL
ON CONFLICT (code) DO NOTHING;

-- Добавляем параметры из событий, если в старых данных есть нестандартные значения.
INSERT INTO process_parameters (code, name, unit, description)
SELECT DISTINCT
    ae.parameter_type,
    ae.parameter_type,
    ae.unit,
    'Автоматически добавленный параметр из существующих событий'
FROM alert_events ae
         LEFT JOIN process_parameters pp ON pp.code = ae.parameter_type
WHERE pp.code IS NULL
  AND ae.parameter_type IS NOT NULL
ON CONFLICT (code) DO NOTHING;

-- Добавляем параметры из аномалий.
-- У аномалий нет поля unit, поэтому используем техническую единицу state.
INSERT INTO process_parameters (code, name, unit, description)
SELECT DISTINCT
    an.parameter_type,
    an.parameter_type,
    'state',
    'Автоматически добавленный параметр из существующих аномалий'
FROM anomaly_events an
         LEFT JOIN process_parameters pp ON pp.code = an.parameter_type
WHERE pp.code IS NULL
  AND an.parameter_type IS NOT NULL
ON CONFLICT (code) DO NOTHING;

-- Добавляем параметры из весов качества, если там есть что-то дополнительное.
INSERT INTO process_parameters (code, name, unit, description)
SELECT DISTINCT
    qw.parameter,
    qw.parameter,
    'weight',
    'Автоматически добавленный параметр из весов индекса качества'
FROM quality_weights qw
         LEFT JOIN process_parameters pp ON pp.code = qw.parameter
WHERE pp.code IS NULL
  AND qw.parameter IS NOT NULL
ON CONFLICT (code) DO NOTHING;

-- Если в старых данных есть ссылки на несуществующих пользователей,
-- перед добавлением FK лучше обнулить их, иначе миграция упадёт.
UPDATE setpoints s
SET updated_by = NULL
WHERE updated_by IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM users u
    WHERE u.id = s.updated_by
);

UPDATE alert_events ae
SET acknowledged_by = NULL
WHERE acknowledged_by IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM users u
    WHERE u.id = ae.acknowledged_by
);

ALTER TABLE telemetry_readings
    ADD CONSTRAINT fk_telemetry_readings_parameter
        FOREIGN KEY (parameter_type)
            REFERENCES process_parameters(code)
            ON UPDATE CASCADE
            ON DELETE RESTRICT;

ALTER TABLE setpoints
    ADD CONSTRAINT fk_setpoints_parameter
        FOREIGN KEY (parameter_type)
            REFERENCES process_parameters(code)
            ON UPDATE CASCADE
            ON DELETE RESTRICT;

ALTER TABLE alert_events
    ADD CONSTRAINT fk_alert_events_parameter
        FOREIGN KEY (parameter_type)
            REFERENCES process_parameters(code)
            ON UPDATE CASCADE
            ON DELETE RESTRICT;

ALTER TABLE anomaly_events
    ADD CONSTRAINT fk_anomaly_events_parameter
        FOREIGN KEY (parameter_type)
            REFERENCES process_parameters(code)
            ON UPDATE CASCADE
            ON DELETE RESTRICT;

ALTER TABLE quality_weights
    ADD CONSTRAINT fk_quality_weights_parameter
        FOREIGN KEY (parameter)
            REFERENCES process_parameters(code)
            ON UPDATE CASCADE
            ON DELETE RESTRICT;

ALTER TABLE setpoints
    ADD CONSTRAINT fk_setpoints_updated_by
        FOREIGN KEY (updated_by)
            REFERENCES users(id)
            ON UPDATE CASCADE
            ON DELETE SET NULL;

ALTER TABLE alert_events
    ADD CONSTRAINT fk_alert_events_acknowledged_by
        FOREIGN KEY (acknowledged_by)
            REFERENCES users(id)
            ON UPDATE CASCADE
            ON DELETE SET NULL;

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