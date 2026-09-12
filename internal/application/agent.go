package application

import (
	"fmt"
	"log/slog"

	"github.com/google/uuid"

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

// GetAgents gets the list of all root agents
func (a *App) GetAgents() ([]schema.Agent, error) {
	rootAgentsList, err := a.agentService.ListRootAgents(a.ctx)
	if err != nil {
		slog.ErrorContext(a.ctx, "Error getting agent list", "error", err)
		return nil, err
	}
	return rootAgentsList, nil
}

func (a *App) GetAgent(id string) (schema.AgentDetail, error) {
	agent, err := a.agentService.GetAgentDetail(a.ctx, uuid.MustParse(id))
	if err != nil {
		return agent, err
	}
	return agent, nil
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
		slog.ErrorContext(a.ctx, "Error getting sub-agents", "error", err)
		return nil, err
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
