BEGIN;

ALTER TABLE sensor_logs
    ADD COLUMN IF NOT EXISTS device_code VARCHAR(100),
    ADD COLUMN IF NOT EXISTS sequence_no BIGINT,
    ADD COLUMN IF NOT EXISTS received_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE TABLE IF NOT EXISTS device_status (
    device_code VARCHAR(100) PRIMARY KEY,
    firmware_version VARCHAR(100),
    last_seen_at TIMESTAMPTZ NOT NULL,
    last_payload_at TIMESTAMPTZ,
    last_ip VARCHAR(64),
    last_sequence_no BIGINT,
    total_requests BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_sensor_logs_device_sequence_plant
    ON sensor_logs (device_code, sequence_no, plant_id)
    WHERE device_code IS NOT NULL
      AND sequence_no IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_sensor_logs_device_received
    ON sensor_logs (device_code, received_at DESC);

CREATE INDEX IF NOT EXISTS idx_device_status_last_seen
    ON device_status (last_seen_at DESC);

COMMIT;