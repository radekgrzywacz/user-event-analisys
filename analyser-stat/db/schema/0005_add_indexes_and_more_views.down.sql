DROP VIEW IF EXISTS v_session_anomalies;
DROP VIEW IF EXISTS v_user_activity_summary;

ALTER TABLE anomalies
DROP COLUMN IF EXISTS probability,
DROP COLUMN IF EXISTS zscore,
DROP COLUMN IF EXISTS ema,
DROP COLUMN IF EXISTS deviation;

DROP INDEX IF EXISTS idx_events_user_timestamp;
DROP INDEX IF EXISTS idx_anomalies_user_detected_at;
DROP INDEX IF EXISTS idx_anomalies_type;
