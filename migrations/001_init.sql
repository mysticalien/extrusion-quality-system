CREATE TABLE IF NOT EXISTS telemetry_readings (
                                                  id BIGSERIAL PRIMARY KEY,
                                                  parameter_type TEXT NOT NULL,
                                                  value DOUBLE PRECISION NOT NULL,
                                                  unit TEXT NOT NULL,
                                                  source_id TEXT NOT NULL,
                                                  measured_at TIMESTAMPTZ NOT NULL,
                                                  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_telemetry_readings_parameter_measured_at
    ON telemetry_readings (parameter_type, measured_at DESC);

CREATE TABLE IF NOT EXISTS alert_events (
                                            id BIGSERIAL PRIMARY KEY,
                                            parameter_type TEXT NOT NULL,
                                            level TEXT NOT NULL,
                                            status TEXT NOT NULL,
                                            value DOUBLE PRECISION NOT NULL,
                                            unit TEXT NOT NULL,
                                            source_id TEXT NOT NULL,
                                            message TEXT NOT NULL,
                                            created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                            acknowledged_at TIMESTAMPTZ,
                                            acknowledged_by BIGINT,
                                            resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_alert_events_status_created_at
    ON alert_events (status, created_at DESC);

CREATE TABLE IF NOT EXISTS quality_index_values (
                                                    id BIGSERIAL PRIMARY KEY,
                                                    value DOUBLE PRECISION NOT NULL,
                                                    state TEXT NOT NULL,
                                                    parameter_penalty DOUBLE PRECISION NOT NULL,
                                                    anomaly_penalty DOUBLE PRECISION NOT NULL,
                                                    calculated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_quality_index_values_calculated_at
    ON quality_index_values (calculated_at DESC);