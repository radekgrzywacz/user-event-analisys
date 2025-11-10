CREATE INDEX idx_events_session_id ON events(session_id);

CREATE OR REPLACE VIEW v_user_sessions AS
SELECT
  session_id,
  user_id,
  MIN(timestamp) AS start_time,
  MAX(timestamp) AS end_time,
  COUNT(*) AS events,
  BOOL_OR(event_type = 'payment') AS has_payment,
  SUM(CASE WHEN event_type = 'payment' THEN (additional->>'value')::float ELSE 0 END) AS total_value
FROM events
WHERE session_id IS NOT NULL
GROUP BY session_id, user_id
ORDER BY start_time DESC;

CREATE OR REPLACE VIEW v_anomalies_details AS
SELECT
  a.id,
  a.user_id,
  e.event_type AS event_type,
  e.timestamp,
  e.session_id,
  e.additional->>'source' AS source,
  e.additional->>'value' AS value,
  a.anomaly_type,
  a.details->>'message' AS message
FROM anomalies a
LEFT JOIN events e ON e.id = a.event_id
ORDER BY a.detected_at DESC;
