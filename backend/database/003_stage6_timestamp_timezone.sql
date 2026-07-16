BEGIN;

ALTER TABLE public.sensor_logs
    ALTER COLUMN recorded_at DROP DEFAULT;

ALTER TABLE public.sensor_logs
    ALTER COLUMN recorded_at TYPE timestamptz
    USING recorded_at AT TIME ZONE 'Asia/Jakarta';

ALTER TABLE public.sensor_logs
    ALTER COLUMN recorded_at SET DEFAULT NOW();

ALTER TABLE public.sensor_logs
    ALTER COLUMN received_at SET DEFAULT NOW();

COMMIT;