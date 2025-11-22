-- name: InsertMLResult :one
INSERT INTO ml_results (
    user_id,
    session_id,
    timestamp,
    anomaly,
    score,
    threshold,
    event_count,
    unique_events,
    source
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING id;

-- name: GetMLResultBySession :one
SELECT *
FROM ml_results
WHERE session_id = $1
ORDER BY timestamp DESC
LIMIT 1;
