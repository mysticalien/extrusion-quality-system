CREATE TABLE IF NOT EXISTS anomaly_events (
                                              id BIGSERIAL PRIMARY KEY,

                                              type TEXT NOT NULL,
                                              parameter_type TEXT NOT NULL,
                                              level TEXT NOT NULL,
                                              status TEXT NOT NULL,

                                              message TEXT NOT NULL,

                                              current_value DOUBLE PRECISION,
                                              previous_value DOUBLE PRECISION,

                                              source_id TEXT NOT NULL,
                                              observed_at TIMESTAMPTZ NOT NULL,

                                              created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                              updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                              resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_anomaly_events_status_created_at
    ON anomaly_events (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_anomaly_events_type_parameter_status
    ON anomaly_events (type, parameter_type, status);