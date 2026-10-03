-- name: GetMCPOAuthClientConfig :one
SELECT mcp_server_id,
       auth_url,
       token_url,
       client_id,
       client_secret,
       scopes,
       auth_style,
       created_at,
       updated_at
FROM mcp_oauth_client_config
WHERE mcp_server_id = sqlc.arg('mcp_server_id');

-- name: UpsertMCPOAuthClientConfig :one
INSERT INTO mcp_oauth_client_config (mcp_server_id, auth_url, token_url, client_id, client_secret, scopes, auth_style)
VALUES (sqlc.arg('mcp_server_id'),
        sqlc.arg('auth_url'),
        sqlc.arg('token_url'),
        sqlc.arg('client_id'),
        sqlc.narg('client_secret'),
        sqlc.arg('scopes'),
        sqlc.arg('auth_style'))
ON CONFLICT (mcp_server_id) DO UPDATE
    SET auth_url      = excluded.auth_url,
        token_url     = excluded.token_url,
        client_id     = excluded.client_id,
        client_secret = excluded.client_secret,
        scopes        = excluded.scopes,
        auth_style    = excluded.auth_style,
        updated_at    = datetime('now')
RETURNING mcp_server_id, auth_url, token_url, client_id, client_secret, scopes, auth_style, created_at, updated_at;

-- name: GetMCPOAuthGrant :one
SELECT user_id,
       mcp_server_id,
       access_token,
       refresh_token,
       expiry,
       created_at,
       updated_at
FROM mcp_oauth_grant
WHERE user_id = sqlc.arg('user_id')
  AND mcp_server_id = sqlc.arg('mcp_server_id');

-- name: UpsertMCPOAuthGrant :one
INSERT INTO mcp_oauth_grant (user_id, mcp_server_id, access_token, refresh_token, expiry)
VALUES (sqlc.arg('user_id'),
        sqlc.arg('mcp_server_id'),
        sqlc.arg('access_token'),
        sqlc.narg('refresh_token'),
        sqlc.narg('expiry'))
ON CONFLICT (user_id, mcp_server_id) DO UPDATE
    SET access_token  = excluded.access_token,
        refresh_token = excluded.refresh_token,
        expiry        = excluded.expiry,
        updated_at    = datetime('now')
RETURNING user_id, mcp_server_id, access_token, refresh_token, expiry, created_at, updated_at;

-- name: DeleteMCPOAuthGrant :exec
DELETE
FROM mcp_oauth_grant
WHERE user_id = sqlc.arg('user_id')
  AND mcp_server_id = sqlc.arg('mcp_server_id');