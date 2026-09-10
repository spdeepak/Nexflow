-- name: CreateRun :one
INSERT INTO runs (id, session_id, invocation_id, root_agent_id)
VALUES (?1, ?2, ?3, ?4)
RETURNING id, session_id, invocation_id, root_agent_id, status, started_at, finished_at, error;

-- name: GetRun :one
SELECT id,
       session_id,
       invocation_id,
       root_agent_id,
       status,
       started_at,
       finished_at,
       error
FROM runs
WHERE id = ?1;

-- name: ListRuns :many
SELECT id,
       session_id,
       invocation_id,
       root_agent_id,
       status,
       started_at,
       finished_at,
       error
FROM runs
WHERE session_id = ?1
  AND (?2 IS NULL OR status = ?2)
ORDER BY started_at DESC;

-- name: UpdateRunStatus :one
UPDATE runs
SET status         = ?2,
    finished_at    = CASE WHEN ?2 IN ('completed', 'failed', 'interrupted') THEN datetime('now') ELSE finished_at END,
    error          = COALESCE(?3, error)
WHERE id = ?1
RETURNING id, session_id, invocation_id, root_agent_id, status, started_at, finished_at, error;

-- name: DeleteSessionRuns :exec
DELETE
FROM runs
WHERE session_id = ?1;
