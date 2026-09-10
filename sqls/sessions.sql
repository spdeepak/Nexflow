-- name: CreateSession :one
INSERT INTO sessions (id, user_id, app_name, root_agent_id)
VALUES (?1, ?2, ?3, ?4)
RETURNING id, user_id, app_name, root_agent_id, last_update, created_at;

-- name: GetSession :one
SELECT id, user_id, app_name, root_agent_id, last_update, created_at
FROM sessions
WHERE id = ?1;

-- name: ListSessions :many
SELECT id, user_id, app_name, root_agent_id, last_update, created_at
FROM sessions
WHERE (CAST(sqlc.narg('user_id') as UUID) IS NULL OR user_id = sqlc.narg('user_id'))
  AND (app_name = sqlc.arg('app_name'))
ORDER BY last_update DESC;

-- name: UpdateSessionLastUpdate :exec
UPDATE sessions
SET last_update = datetime('now')
WHERE id = ?1;

-- name: DeleteSession :exec
DELETE
FROM sessions
WHERE id = ?1;
