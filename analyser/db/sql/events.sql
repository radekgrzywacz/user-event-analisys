-- name: InsertEvent :one
INSERT INTO events (
    user_id,
    event_type,
    timestamp,
    ip,
    user_agent,
    country,
    metadata
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: GetEventById :one
SELECT id,
       user_id,
       event_type,
       timestamp,
       ip,
       user_agent,
       country,
       metadata,
       created_at
FROM events
WHERE id = $1;

-- name: GetEventsByUser :many
SELECT id,
       user_id,
       event_type,
       timestamp,
       ip,
       user_agent,
       country,
       metadata,
       created_at
FROM events
WHERE user_id = $1
ORDER BY timestamp DESC
LIMIT $2;

-- name: GetAllEvents :many
SELECT id,
       user_id,
       event_type,
       timestamp,
       ip,
       user_agent,
       country,
       metadata,
       created_at
FROM events
ORDER BY timestamp DESC
LIMIT $1;

-- name: DeleteOldEvents :exec
DELETE FROM events
WHERE timestamp < NOW() - INTERVAL '7 days';
