-- name: CreateToolCall :one
INSERT INTO tool_calls (event_id, run_id, agent_name, tool_name, arguments, result_json, status, duration_ms)
VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8)
RETURNING id, event_id, run_id, agent_name, tool_name, arguments, result_json, status, duration_ms, created_at;

-- name: ListToolCalls :many
SELECT id,
       event_id,
       run_id,
       agent_name,
       tool_name,
       arguments,
       result_json,
       status,
       duration_ms,
       created_at
FROM tool_calls
WHERE run_id = ?1
  AND (?2 IS NULL OR tool_name = ?2)
  AND (?3 IS NULL OR status = ?3)
ORDER BY created_at ASC;
