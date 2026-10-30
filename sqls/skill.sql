-- name: CreateSkill :one
-- user_id must be NULL for app scope (skills_owner_check), so the DB zeroes
-- it here rather than trusting every caller to remember.
INSERT INTO skills (id, user_id, scope, title, content_type, content, storage_uri, global_skill, metadata)
VALUES (sqlc.arg('id'),
        CASE WHEN sqlc.arg('scope') = 'app' THEN NULL ELSE sqlc.arg('user_id') END,
        sqlc.arg('scope'), sqlc.arg('title'), sqlc.arg('content_type'),
        sqlc.arg('content'), sqlc.arg('storage_uri'), sqlc.arg('global_skill'), sqlc.arg('metadata'))
RETURNING id, user_id, scope, title, content_type, content, storage_uri, metadata, is_active, created_at, updated_at;

-- name: GetSkill :one
SELECT id,
       user_id,
       scope,
       title,
       content_type,
       content,
       storage_uri,
       global_skill,
       metadata,
       is_active,
       created_at,
       updated_at
FROM skills
WHERE id = ?1;

-- name: GetAvailableSkill :many
-- Ownership filter: every user sees app-scope skills plus their own.
SELECT id,
       user_id,
       scope,
       title,
       content_type,
       content,
       storage_uri,
       global_skill,
       metadata,
       is_active,
       created_at,
       updated_at,
       (SELECT COUNT(*) FROM agent_skills WHERE skill_id = skills.id) AS agent_count
FROM skills
WHERE scope = 'app'
   OR user_id = sqlc.arg('user_id')
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
RETURNING id, user_id, scope, title, content_type, content, storage_uri, global_skill, metadata, is_active, created_at, updated_at;

-- name: DeleteSkill :execresult
DELETE
FROM skills
WHERE id = sqlc.arg('id')
  AND (scope = 'app' OR user_id = sqlc.arg('user_id'));

-- name: GetSkillByTitle :one
SELECT id,
       user_id,
       scope,
       title,
       content_type,
       content,
       storage_uri,
       global_skill,
       metadata,
       is_active,
       created_at,
       updated_at
FROM skills
WHERE title = ?1;

-- name: GetSkillsByContentTypeAndScope :many
SELECT id,
       user_id,
       scope,
       title,
       content_type,
       content,
       storage_uri,
       global_skill,
       metadata,
       is_active,
       created_at,
       updated_at
FROM skills
WHERE content_type = ?1
  AND scope = ?2;
