-- name: CreateUserModelCredential :one
INSERT INTO model_credentials (id, user_id, provider, model_name, title, base_url, scope, api_key_cipher, key_version)
VALUES (sqlc.arg('id'),
        sqlc.arg('user_id'),
        sqlc.arg('provider'),
        sqlc.arg('model_name'),
        sqlc.arg('title'),
        sqlc.arg('base_url'),
        'user',
        sqlc.arg('api_key_cipher'),
        sqlc.arg('key_version'))
RETURNING id, user_id, provider, model_name, title, base_url, scope, key_version, is_active, created_at, updated_at;

-- name: CreateAppModelCredential :one
INSERT INTO model_credentials (id, provider, model_name, title, base_url, scope, api_key_cipher, key_version)
VALUES (sqlc.arg('id'),
        sqlc.arg('provider'),
        sqlc.arg('model_name'),
        sqlc.arg('title'),
        sqlc.arg('base_url'),
        'app',
        sqlc.arg('api_key_cipher'),
        sqlc.arg('key_version'))
RETURNING id, provider, model_name, title, base_url, scope, key_version, is_active, created_at, updated_at;

-- name: GetUserModelCredentials :many
SELECT id,
       user_id,
       provider,
       model_name,
       title,
       base_url,
       scope,
       key_version,
       is_active,
       created_at,
       updated_at
FROM model_credentials
WHERE user_id = @user_id
  AND is_active = COALESCE(sqlc.narg('is_active'), is_active)
ORDER BY created_at DESC;

-- name: GetAvailableModelCredentials :many
SELECT id,
       user_id,
       provider,
       model_name,
       title,
       base_url,
       scope,
       key_version,
       is_active,
       created_at,
       updated_at
FROM model_credentials
WHERE (scope = 'app' OR user_id = @user_id)
  AND (sqlc.narg('is_active') IS NULL OR is_active = sqlc.narg('is_active'))
ORDER BY scope, created_at DESC;

-- name: GetModelCredential :one
SELECT id,
       user_id,
       provider,
       model_name,
       title,
       base_url,
       scope,
       key_version,
       is_active,
       created_at,
       updated_at
FROM model_credentials
WHERE id = ?1;

-- name: GetModelCredentialWithKey :one
SELECT id,
       user_id,
       provider,
       model_name,
       title,
       base_url,
       scope,
       api_key_cipher,
       key_version,
       is_active,
       created_at,
       updated_at
FROM model_credentials
WHERE id = ?1;

-- name: UpdateModelCredential :one
UPDATE model_credentials
SET api_key_cipher = COALESCE(sqlc.narg('api_key_cipher'), api_key_cipher),
    provider       = COALESCE(sqlc.narg('provider'), provider),
    model_name     = COALESCE(sqlc.narg('model_name'), model_name),
    title          = COALESCE(sqlc.narg('title'), title),
    base_url       = COALESCE(sqlc.narg('base_url'), base_url),
    is_active      = COALESCE(sqlc.narg('is_active'), is_active),
    key_version    = CASE WHEN sqlc.narg('api_key_cipher') IS NOT NULL THEN key_version + 1 ELSE key_version END,
    updated_at     = datetime('now')
WHERE id = sqlc.arg('id')
RETURNING id, user_id, provider, model_name, title, base_url, scope, key_version, is_active, created_at, updated_at;

-- name: DeleteModelCredential :exec
DELETE
FROM model_credentials
WHERE id = ?1;
