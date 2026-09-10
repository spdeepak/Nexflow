-- name: CreateArtifact :one
INSERT INTO artifacts (session_id, filename, version, agent_name, storage_uri, size_bytes, content_type)
VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7)
RETURNING id, session_id, filename, version, agent_name, storage_uri, size_bytes, content_type, created_at;

-- name: ListArtifacts :many
SELECT id,
       session_id,
       filename,
       version,
       agent_name,
       storage_uri,
       size_bytes,
       content_type,
       created_at
FROM artifacts
WHERE session_id = ?1
ORDER BY filename, version DESC;

-- name: GetArtifact :one
SELECT id,
       session_id,
       filename,
       version,
       agent_name,
       storage_uri,
       size_bytes,
       content_type,
       created_at
FROM artifacts
WHERE session_id = ?1
  AND id = ?2;

-- name: DeleteSessionArtifacts :exec
DELETE
FROM artifacts
WHERE session_id = ?1;
