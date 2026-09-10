-- name: LinkAgentAndMCP :exec
INSERT INTO agent_mcp_server (agent_id, mcp_server_id)
VALUES (sqlc.arg('agent_id'),
        sqlc.arg('mcp_server_id'));

-- name: ListAgentMCP :many
SELECT m.id,
       m.user_id,
       m.name,
       m.endpoint,
       m.transport,
       m.command,
       m.args,
       m.auth_type,
       m.auth_config,
       m.allowed_tools,
       m.require_confirmation,
       m.confirmation_rules,
       m.is_active,
       m.created_at,
       m.updated_at
FROM mcp_server m
         JOIN agent_mcp_server am ON am.mcp_server_id = m.id
WHERE am.agent_id = sqlc.arg('agent_id')
ORDER BY m.name;


-- name: DetachMCPFromAgent :exec
DELETE
FROM agent_mcp_server
WHERE agent_id = sqlc.arg('agent_id')
  AND mcp_server_id = sqlc.arg('mcp_server_id');
