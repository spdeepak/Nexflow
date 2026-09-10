-- name: CreateSkill :one
INSERT INTO skills (id, user_id, scope, title, content_type, content, storage_uri, metadata)
VALUES (sqlc.arg('id'), sqlc.arg('user_id'), sqlc.arg('scope'), sqlc.arg('title'), sqlc.arg('content_type'),
        sqlc.arg('content'), sqlc.arg('storage_uri'), sqlc.arg('metadata'))
RETURNING id, user_id, scope, title, content_type, content, storage_uri, metadata, is_active, created_at, updated_at;

-- name: GetSkill :one
SELECT id,
       user_id,
       scope,
       title,
       content_type,
       content,
       storage_uri,
       metadata,
       is_active,
       created_at,
       updated_at
FROM skills
WHERE id = ?1;

-- name: GetAvailableSkill :many
SELECT id,
       user_id,
       scope,
       title,
       content_type,
       content,
       storage_uri,
       metadata,
       is_active,
       created_at,
       updated_at,
       (SELECT COUNT(*) FROM agent_skills WHERE skill_id = skills.id) AS agent_count
FROM skills
WHERE (scope = 'app' OR user_id = sqlc.narg('user_id'))
  AND (CAST(sqlc.narg('is_active') as BOOLEAN) IS NULL OR is_active = sqlc.narg('is_active'))
  AND (scope = sqlc.narg('scope') IS NULL OR scope = sqlc.narg('scope'))
ORDER BY created_at DESC;

-- name: GetAllAvailableSkill :many
SELECT id,
       user_id,
       scope,
       title,
       content_type,
       content,
       storage_uri,
       metadata,
       is_active,
       created_at,
       updated_at,
       (SELECT COUNT(*) FROM agent_skills WHERE skill_id = skills.id) AS agent_count
FROM skills
ORDER BY created_at DESC;

-- name: UpdateSkill :one
UPDATE skills
SET title        = COALESCE(sqlc.narg('title'), title),
    content_type = COALESCE(sqlc.narg('content_type'), content_type),
    content      = COALESCE(sqlc.narg('content'), content),
    storage_uri  = COALESCE(sqlc.narg('storage_uri'), storage_uri),
    is_active    = COALESCE(sqlc.narg('is_active'), is_active),
    metadata     = COALESCE(sqlc.narg('metadata'), metadata),
    updated_at   = datetime('now')
WHERE id = sqlc.arg('id')
RETURNING id, user_id, scope, title, content_type, content, storage_uri, metadata, is_active, created_at, updated_at;

-- name: DeleteSkill :exec
DELETE
FROM skills
WHERE id = sqlc.arg('id')
  and user_id = sqlc.arg('user_id');
