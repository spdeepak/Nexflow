package application

import (
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/agentmcp"
	"github.com/spdeepak/nexflow/internal/agentskills"
	"github.com/spdeepak/nexflow/internal/schema"
)

func (a *App) CreateAgent(params schema.AgentCreate) error {
	if err := params.Validate(); err != nil {
		return err
	}
	_, err := a.agentService.CreateRootAgent(a.ctx, params)
	return err
}

func (a *App) CreateSubAgent(params schema.AgentCreate) error {
	if err := params.Validate(); err != nil {
		return err
	}
	_, err := a.agentService.CreateSubAgent(a.ctx, params)
	return err
}

func (a *App) GetRootAgents() ([]schema.Agent, error) {
	rootAgentsList, err := a.agentService.ListRootAgents(a.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list root agents: %w", err)
	}
	return rootAgentsList, nil
}

func (a *App) GetAgent(id string) (schema.AgentDetail, error) {
	return a.agentService.GetAgentDetail(a.ctx, uuid.MustParse(id))
}

func (a *App) DeleteAgent(id string) error {
	return a.agentService.DeleteAgent(a.ctx, uuid.MustParse(id))
}

func (a *App) UpdateAgent(id string, params schema.AgentUpdate) error {
	if err := params.Validate(); err != nil {
		return err
	}
	agentID, err := uuid.Parse(id)
	if err != nil {
		slog.ErrorContext(a.ctx, "Error parsing agent id", "agent_id", id, "error", err)
		return fmt.Errorf("invalid agent ID: %w", err)
	}

	_, err = a.agentService.UpdateAgent(a.ctx, agentID, params)
	return err
}

func (a *App) GetSubAgents(parentId string) ([]schema.Agent, error) {
	children, err := a.agentService.GetSubAgents(a.ctx, uuid.MustParse(parentId))
	if err != nil {
		return nil, fmt.Errorf("failed to list sub-agents: %w", err)
	}
	return children, nil
}

func (a *App) ReorderSubAgents(parentId string, orderedIDs []string) error {
	parentUUID, err := uuid.Parse(parentId)
	if err != nil {
		return fmt.Errorf("invalid parent agent ID: %w", err)
	}
	ids := make([]uuid.UUID, len(orderedIDs))
	for i, id := range orderedIDs {
		parsed, err := uuid.Parse(id)
		if err != nil {
			slog.ErrorContext(a.ctx, "Error parsing sub-agent id for reorder", "sub_agent_id", id, "error", err)
			return fmt.Errorf("invalid sub-agent ID: %w", err)
		}
		ids[i] = parsed
	}
	return a.agentService.ReorderAgentChildren(a.ctx, parentUUID, ids)
}

// parseAgentRelation parses both IDs and confirms the agent exists before a
// relation is written. relatedKind only labels the error message.
func (a *App) parseAgentRelation(agentId, relatedId, relatedKind string) (uuid.UUID, uuid.UUID, error) {
	agentID, err := uuid.Parse(agentId)
	if err != nil {
		slog.ErrorContext(a.ctx, "Error parsing agent id", "agent_id", agentId, "error", err)
		return uuid.UUID{}, uuid.UUID{}, fmt.Errorf("invalid agent ID: %w", err)
	}
	relatedUUID, err := uuid.Parse(relatedId)
	if err != nil {
		slog.ErrorContext(a.ctx, "Error parsing related id", "id", relatedId, "error", err)
		return uuid.UUID{}, uuid.UUID{}, fmt.Errorf("invalid %s ID: %w", relatedKind, err)
	}
	if _, err := a.agentService.GetAgent(a.ctx, agentID); err != nil {
		return uuid.UUID{}, uuid.UUID{}, fmt.Errorf("failed to load agent %s: %w", agentId, err)
	}
	return agentID, relatedUUID, nil
}

func (a *App) AttachSkillToAgent(agentId string, skillId string) error {
	agentID, skillUUID, err := a.parseAgentRelation(agentId, skillId, "skill")
	if err != nil {
		return err
	}
	return a.agentSkillService.AttachSkillToAgent(a.ctx, agentskills.AttachSkillToAgentParams{
		AgentID: agentID,
		SkillID: skillUUID,
	})
}

func (a *App) DetachSkillFromAgent(agentId string, skillId string) error {
	agentID, skillUUID, err := a.parseAgentRelation(agentId, skillId, "skill")
	if err != nil {
		return err
	}
	return a.agentSkillService.DetachSkillFromAgent(a.ctx, agentskills.DetachSkillFromAgentParams{
		AgentID: agentID,
		SkillID: skillUUID,
	})
}

func (a *App) AttachMCPToAgent(agentId string, mcpId string) error {
	agentID, mcpUUID, err := a.parseAgentRelation(agentId, mcpId, "mcp")
	if err != nil {
		return err
	}
	return a.agentMCPService.LinkAgentAndMCP(a.ctx, agentmcp.LinkAgentAndMCPParams{
		AgentID:     agentID,
		McpServerID: mcpUUID,
	})
}

func (a *App) DetachMCPFromAgent(agentId string, mcpId string) error {
	agentID, mcpUUID, err := a.parseAgentRelation(agentId, mcpId, "mcp")
	if err != nil {
		return err
	}
	return a.agentMCPService.DetachMCPFromAgent(a.ctx, agentmcp.DetachMCPFromAgentParams{
		AgentID:     agentID,
		McpServerID: mcpUUID,
	})
}
