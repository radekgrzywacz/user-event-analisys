-- name: InsertAggregatedResult :one
INSERT INTO aggregated_results (
    session_id,
    user_id,
    ml_anomaly,
    ml_score,
    ml_threshold,
    stat_anomaly,
    anomaly_type,
    event_count,
    unique_events,
    created_at
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, now()
)
RETURNING id;

-- name: GetAggregatedResult :one
SELECT *
FROM aggregated_results
WHERE session_id = $1
ORDER BY created_at DESC
LIMIT 1;
