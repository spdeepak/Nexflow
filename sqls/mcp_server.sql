-- name: CreateMCPServer :one
INSERT INTO mcp_server (id, user_id, name, endpoint, transport, command, args, auth_type, auth_config, allowed_tools,
                        require_confirmation, confirmation_rules, is_active)
VALUES (sqlc.arg('id'),
        sqlc.arg('user_id'),
        sqlc.arg('name'),
        sqlc.arg('endpoint'),
        sqlc.arg('transport'),
        sqlc.narg('command'),
        sqlc.narg('args'),
        sqlc.narg('auth_type'),
        sqlc.narg('auth_config'),
        sqlc.narg('allowed_tools'),
        sqlc.arg('require_confirmation'),
        sqlc.narg('confirmation_rules'),
        sqlc.arg('is_active'))
RETURNING id, user_id, name, endpoint, transport, command, args, auth_type, auth_config, allowed_tools,
    require_confirmation, confirmation_rules, is_active, created_at, updated_at;

-- name: ListMCPServers :many
SELECT id,
       user_id,
       name,
       endpoint,
       transport,
       command,
       args,
       auth_type,
       auth_config,
       allowed_tools,
       require_confirmation,
       confirmation_rules,
       is_active,
       created_at,
       updated_at
FROM mcp_server
ORDER BY created_at DESC;

-- name: UpdateMCPServer :one
UPDATE mcp_server
SET name                 = COALESCE(sqlc.narg('name'), name),
    endpoint             = COALESCE(sqlc.narg('endpoint'), endpoint),
    transport            = COALESCE(sqlc.narg('transport'), transport),
    command              = COALESCE(sqlc.narg('command'), command),
    args                 = COALESCE(sqlc.narg('args'), args),
    auth_type            = COALESCE(sqlc.narg('auth_type'), auth_type),
    auth_config          = COALESCE(sqlc.narg('auth_config'), auth_config),
    allowed_tools        = COALESCE(sqlc.narg('allowed_tools'), allowed_tools),
    require_confirmation = COALESCE(sqlc.narg('require_confirmation'), require_confirmation),
    confirmation_rules   = COALESCE(sqlc.narg('confirmation_rules'), confirmation_rules),
    is_active            = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at           = datetime('now')
WHERE id = sqlc.arg('id')
RETURNING id, user_id, name, endpoint, transport, command, args, auth_type, auth_config, allowed_tools,
    require_confirmation, confirmation_rules, is_active, created_at, updated_at;;

-- name: DeleteMCPServer :exec
DELETE
FROM mcp_server
where id = sqlc.arg('id');