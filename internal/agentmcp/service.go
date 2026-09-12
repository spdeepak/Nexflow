package agentmcp

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/schema"
)

type (
	service struct {
		querier Querier
	}

	Service interface {
		DetachMCPFromAgent(ctx context.Context, arg DetachMCPFromAgentParams) error
		LinkAgentAndMCP(ctx context.Context, arg LinkAgentAndMCPParams) error
		ListAgentMCP(ctx context.Context, agentID uuid.UUID) ([]schema.MCP, error)
	}
)

func NewService(querier Querier) Service {
	return &service{
		querier: querier,
	}
}

func (s *service) LinkAgentAndMCP(ctx context.Context, arg LinkAgentAndMCPParams) error {
	err := s.querier.LinkAgentAndMCP(ctx, arg)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to link agent to MCP", "mcp_id", arg.McpServerID, "agent_id", arg.AgentID)
		return err
	}
	return nil
}

func (s *service) DetachMCPFromAgent(ctx context.Context, arg DetachMCPFromAgentParams) error {
	err := s.querier.DetachMCPFromAgent(ctx, arg)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to unlink agent to MCP", "mcp_id", arg.McpServerID, "agent_id", arg.AgentID)
		return err
	}
	return nil
}

func (s *service) ListAgentMCP(ctx context.Context, agentID uuid.UUID) ([]schema.MCP, error) {
	mcpList, err := s.querier.ListAgentMCP(ctx, agentID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list agent mcp", "agent_id", agentID)
		return nil, err
	}
	schemaMCPs := make([]schema.MCP, len(mcpList))
	for i, mcp := range mcpList {
		if schemaMcp, err := convertDbMCPToSchemaMCP(mcp); err == nil {
			schemaMCPs[i] = schemaMcp
		}
	}
	return schemaMCPs, nil
}

func convertDbMCPToSchemaMCP(mcpServer McpServer) (schema.MCP, error) {
	var authConfig schema.AuthConfig
	if mcpServer.AuthConfig != nil {
		if err := json.Unmarshal(mcpServer.AuthConfig, &authConfig); err != nil {
			return schema.MCP{}, err
		}
	}
	var confirmationRules schema.ConfirmationRules
	if mcpServer.ConfirmationRules != nil {
		if err := json.Unmarshal(mcpServer.ConfirmationRules, &confirmationRules); err != nil {
			return schema.MCP{}, err
		}
	}
	return schema.MCP{
		AllowedTools:        unmarshalStringSlice(mcpServer.AllowedTools),
		Args:                unmarshalStringSlice(mcpServer.Args),
		AuthConfig:          authConfig,
		AuthType:            &mcpServer.AuthType,
		Command:             &mcpServer.Command,
		ConfirmationRules:   confirmationRules,
		CreatedAt:           mcpServer.CreatedAt,
		Endpoint:            &mcpServer.Endpoint,
		ID:                  mcpServer.ID,
		IsActive:            mcpServer.IsActive,
		Name:                mcpServer.Name,
		RequireConfirmation: mcpServer.RequireConfirmation,
		Transport:           mcpServer.Transport,
		UpdatedAt:           mcpServer.UpdatedAt,
		UserID:              mcpServer.UserID,
	}, nil
}

func unmarshalStringSlice(s string) []string {
	var result []string
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil
	}
	return result
}
