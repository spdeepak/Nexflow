-- name: AttachSkillToAgent :exec
INSERT INTO agent_skills (agent_id, skill_id)
VALUES (?1, ?2)
ON CONFLICT (agent_id, skill_id) DO NOTHING;

-- name: ListAgentSkill :many
SELECT k.id,
       k.user_id,
       k.scope,
       k.title,
       k.content_type,
       k.content,
       k.storage_uri,
       k.metadata,
       k.is_active,
       k.created_at,
       k.updated_at
FROM skills k
         JOIN agent_skills ak ON ak.skill_id = k.id
WHERE ak.agent_id = ?1
ORDER BY k.title;

-- name: DetachSkillFromAgent :exec
DELETE
FROM agent_skills
WHERE agent_id = ?1
  AND skill_id = ?2;
