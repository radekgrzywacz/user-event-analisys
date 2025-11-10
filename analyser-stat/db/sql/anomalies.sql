-- name: InsertAnomaly :one
INSERT INTO anomalies (
    user_id,
    event_id,
    anomaly_type,
    details,
    detected_at
)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: GetAnomaliesByUser :many
SELECT id,
       user_id,
       event_id,
       anomaly_type,
       details,
       detected_at
FROM anomalies
WHERE user_id = $1
ORDER BY detected_at DESC
LIMIT $2;

-- name: GetAnomaliesByEvent :many
SELECT id,
       user_id,
       event_id,
       anomaly_type,
       details,
       detected_at
FROM anomalies
WHERE event_id = $1;
