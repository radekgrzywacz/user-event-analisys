-- name: InsertStatResult :one
INSERT INTO stat_results (
    user_id,
    session_id,
    event_type,
    anomaly,
    anomaly_type,
    message,
    timestamp,
    source
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING id;

-- name: GetStatResultBySession :one
SELECT *
FROM stat_results
WHERE session_id = $1
ORDER BY timestamp DESC
LIMIT 1;


