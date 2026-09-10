-- name: CreateEvent :one
INSERT INTO events (id, session_id, invocation_id, seq, branch, isolation_scope, author, role, content_json, actions_json,
                    is_partial, is_final, token_usage, output_json)
VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13, ?14)
RETURNING id, session_id, invocation_id, seq, branch, isolation_scope, author, role, content_json, actions_json, is_partial, is_final, token_usage, output_json, created_at;

-- name: GetSessionEvent :one
SELECT id,
       session_id,
       invocation_id,
       seq,
       branch,
       isolation_scope,
       author,
       role,
       content_json,
       actions_json,
       is_partial,
       is_final,
       token_usage,
       output_json,
       created_at
FROM events
WHERE session_id = ?1
  AND id = ?2;

-- name: ListSessionEvents :many
SELECT id,
       session_id,
       invocation_id,
       seq,
       branch,
       isolation_scope,
       author,
       role,
       content_json,
       actions_json,
       is_partial,
       is_final,
       token_usage,
       output_json,
       created_at
FROM events
WHERE session_id = sqlc.arg('session_id')
  AND (sqlc.narg('invocation_id') IS NULL OR invocation_id = sqlc.narg('invocation_id'))
  AND (sqlc.narg('author') IS NULL OR author = sqlc.narg('author'))
  AND (sqlc.narg('role') IS NULL OR role = sqlc.narg('role'))
ORDER BY seq ASC
LIMIT ?1 OFFSET ?2;

-- name: CountSessionEvents :one
SELECT COUNT(*)
FROM events
WHERE session_id = ?1
  AND (?2 IS NULL OR invocation_id = ?2)
  AND (?3 IS NULL OR author = ?3)
  AND (?4 IS NULL OR role = ?4);

-- name: ListRunEvents :many
SELECT id,
       session_id,
       invocation_id,
       seq,
       branch,
       isolation_scope,
       author,
       role,
       content_json,
       actions_json,
       is_partial,
       is_final,
       token_usage,
       output_json,
       created_at
FROM events
WHERE invocation_id = ?1
ORDER BY seq ASC;

-- name: NextEventSeq :one
SELECT CAST(COALESCE(MAX(seq) + 1, 0) AS INTEGER)
FROM events
WHERE session_id = ?1;

-- name: DeleteSessionEvents :exec
DELETE
FROM events
WHERE session_id = ?1;
