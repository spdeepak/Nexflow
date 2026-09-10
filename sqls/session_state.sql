-- name: GetSessionState :many
SELECT session_id, key, value, updated_at
FROM session_state
WHERE session_id = ?1;

-- name: GetSessionStateKey :one
SELECT session_id, key, value, updated_at
FROM session_state
WHERE session_id = ?1
  AND key = ?2;

-- name: SetSessionState :exec
INSERT INTO session_state (session_id, key, value)
VALUES (?1, ?2, ?3)
ON CONFLICT (session_id, key)
    DO UPDATE SET value      = ?3,
                  updated_at = datetime('now');

-- name: DeleteSessionStateKey :exec
DELETE
FROM session_state
WHERE session_id = ?1
  AND key = ?2;

-- name: DeleteSessionState :exec
DELETE
FROM session_state
WHERE session_id = ?1;
