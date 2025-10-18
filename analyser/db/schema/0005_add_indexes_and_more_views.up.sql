CREATE INDEX IF NOT EXISTS idx_events_user_timestamp ON events(user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_anomalies_user_detected_at ON anomalies(user_id, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_anomalies_type ON anomalies(anomaly_type);

CREATE OR REPLACE VIEW v_user_activity_summary AS
SELECT
  user_id,
  date_trunc('hour', timestamp) AS hour,
  COUNT(*) AS total_events,
  COUNT(*) FILTER (WHERE event_type = 'login') AS logins,
  COUNT(*) FILTER (WHERE event_type = 'payment') AS payments,
  COUNT(*) FILTER (WHERE event_type = 'failed_login') AS failed_logins
FROM events
GROUP BY user_id, hour
ORDER BY hour DESC;

ALTER TABLE anomalies
ADD COLUMN IF NOT EXISTS probability DOUBLE PRECISION,
ADD COLUMN IF NOT EXISTS zscore DOUBLE PRECISION,
ADD COLUMN IF NOT EXISTS ema DOUBLE PRECISION,
ADD COLUMN IF NOT EXISTS deviation DOUBLE PRECISION;

CREATE OR REPLACE VIEW v_session_anomalies AS
SELECT
  s.session_id,
  s.user_id,
  s.start_time,
  s.end_time,
  COUNT(a.id) AS anomalies_count,
  ARRAY_AGG(a.anomaly_type) AS anomaly_types
FROM v_user_sessions s
LEFT JOIN anomalies a ON s.user_id = a.user_id
  AND a.detected_at BETWEEN s.start_time AND s.end_time
GROUP BY s.session_id, s.user_id, s.start_time, s.end_time
ORDER BY s.start_time DESC;
