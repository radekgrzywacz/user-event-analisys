-- name: InsertUser :one
INSERT INTO users (id, created_at, last_seen)
VALUES ($1, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET last_seen = NOW()
RETURNING id;

-- name: GetUserByID :one
SELECT id, created_at, last_seen
FROM users
WHERE id = $1;
