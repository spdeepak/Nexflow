-- name: CreateRootAgent :one
INSERT INTO agents (id, name, description, instruction, global_instruction, mode, model_name, model_credential_id,
                    credential_source, model_config, config_json)
VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11)
RETURNING id, name, parent_agent_id, description, instruction, global_instruction, mode, model_name, model_credential_id, credential_source, model_config, config_json, is_active, position, created_at, updated_at;

-- name: CreateSubAgent :one
INSERT INTO agents (id, name, parent_agent_id, description, instruction, global_instruction, mode, model_name,
                    model_credential_id, credential_source, model_config, config_json, position)
SELECT ?1,
       ?2,
       ?3,
       ?4,
       ?5,
       ?6,
       ?7,
       ?8,
       ?9,
       ?10,
       ?11,
       ?12,
       COALESCE(MAX(position), -1) + 1
FROM agents
WHERE parent_agent_id = ?3
RETURNING id, name, parent_agent_id, description, instruction, global_instruction, mode, model_name, model_credential_id, credential_source, model_config, config_json, is_active, position, created_at, updated_at;

-- name: ListRootAgents :many
SELECT id,
       name,
       parent_agent_id,
       description,
       instruction,
       global_instruction,
       mode,
       model_name,
       model_credential_id,
       credential_source,
       model_config,
       config_json,
       is_active,
       position,
       created_at,
       updated_at
FROM agents
WHERE parent_agent_id IS NULL
ORDER BY created_at DESC;

-- name: ListAgentChildren :many
SELECT id,
       name,
       parent_agent_id,
       description,
       instruction,
       global_instruction,
       mode,
       model_name,
       model_credential_id,
       credential_source,
       model_config,
       config_json,
       is_active,
       position,
       created_at,
       updated_at
FROM agents
WHERE parent_agent_id = sqlc.arg('parent_agent_id')
ORDER BY position ASC, created_at ASC;

-- name: GetAgent :one
SELECT id,
       name,
       parent_agent_id,
       description,
       instruction,
       global_instruction,
       mode,
       model_name,
       model_credential_id,
       credential_source,
       model_config,
       config_json,
       is_active,
       position,
       created_at,
       updated_at
FROM agents
WHERE id = sqlc.arg('id');

-- name: GetRootAgent :one
SELECT id,
       name,
       description,
       instruction,
       global_instruction,
       mode,
       model_name,
       model_credential_id,
       credential_source,
       model_config,
       config_json,
       is_active,
       position,
       created_at,
       updated_at
FROM agents
WHERE id = sqlc.arg('id') and parent_agent_id IS NULL;

-- name: UpdateAgent :one
UPDATE agents
SET description         = COALESCE(sqlc.narg('description'), description),
    instruction         = COALESCE(sqlc.narg('instruction'), instruction),
    global_instruction  = COALESCE(sqlc.narg('global_instruction'), global_instruction),
    mode                = COALESCE(CAST(sqlc.narg('mode') AS TEXT), mode),
    model_name          = COALESCE(sqlc.narg('model_name'), model_name),
    model_credential_id = COALESCE(CAST(sqlc.narg('model_credential_id') AS TEXT), model_credential_id),
    credential_source   = COALESCE(CAST(sqlc.narg('credential_source') AS TEXT), credential_source),
    model_config        = COALESCE(sqlc.narg('model_config'), model_config),
    config_json         = COALESCE(sqlc.narg('config_json'), config_json),
    is_active           = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at          = datetime('now')
WHERE id = sqlc.arg('id')
RETURNING id, name, parent_agent_id, description, instruction, global_instruction, mode, model_name, model_credential_id, credential_source, model_config, config_json, is_active, position, created_at, updated_at;

-- name: ReorderAgentChildren :exec
UPDATE agents
SET position   = ?2,
    updated_at = datetime('now')
WHERE id = ?1
  AND parent_agent_id = ?3;

-- name: ReorderSubAgents :exec
UPDATE agents
SET position   = (SELECT CAST(json_extract(value, '$.position') AS INTEGER)
                  FROM json_each(CAST(sqlc.arg(assignments) AS TEXT))
                  WHERE json_extract(value, '$.id') = agents.id),
    updated_at = datetime('now')
WHERE agents.parent_agent_id = sqlc.arg(parent_agent_id)
  AND agents.id IN (SELECT json_extract(value, '$.id')
                    FROM json_each(CAST(sqlc.arg(assignments) AS TEXT)));

-- name: DeleteAgent :exec
DELETE
FROM agents
WHERE id = sqlc.arg('id');

-- name: GetAgentsForModel :many
SELECT id, name
FROM agents
WHERE model_credential_id = sqlc.arg('model_id');