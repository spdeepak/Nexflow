package mcpserver

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/util"
	"github.com/spdeepak/nexflow/schema"
)

type (
	service struct {
		querier Querier
	}

	Service interface {
		CreateMCPServer(ctx context.Context, arg schema.MCPCreate) (schema.MCP, error)
		DeleteMCPServer(ctx context.Context, id uuid.UUID) error
		ListMCPServers(ctx context.Context) ([]schema.MCP, error)
		UpdateMCPServer(ctx context.Context, id uuid.UUID, arg schema.MCPUpdate) (schema.MCP, error)
	}
)

func NewService(querier Querier) Service {
	return &service{
		querier: querier,
	}
}

func (s *service) CreateMCPServer(ctx context.Context, arg schema.MCPCreate) (schema.MCP, error) {
	id, _ := uuid.NewV7()
	createMcp := CreateMCPServerParams{
		ID:                  id,
		UserID:              arg.UserID,
		Name:                arg.Name,
		Endpoint:            arg.Endpoint,
		Transport:           arg.Transport,
		Command:             marshalStringSlice(arg.Command),
		Args:                marshalStringSlice(arg.Args),
		AuthConfig:          util.ToRawMessage[schema.MCPCreateAuthConfig](&arg.AuthConfig),
		AllowedTools:        marshalStringSlice(arg.AllowedTools),
		RequireConfirmation: arg.RequireConfirmation,
		ConfirmationRules:   util.ToRawMessage[schema.MCPCreateConfirmationRules](&arg.ConfirmationRules),
		IsActive:            arg.IsActive,
	}
	if arg.AuthType != nil {
		createMcp.AuthType = *arg.AuthType
	}
	createdMCP, err := s.querier.CreateMCPServer(ctx, createMcp)
	if err != nil {
		return schema.MCP{}, err
	}
	return convertDbMCPToSchemaMCP(createdMCP)
}

func (s *service) DeleteMCPServer(ctx context.Context, id uuid.UUID) error {
	if err := s.querier.DeleteMCPServer(ctx, id); err != nil {
		slog.ErrorContext(ctx, "error deleting MCP server", "id", id)
		return err
	}
	return nil
}

func (s *service) ListMCPServers(ctx context.Context) ([]schema.MCP, error) {
	mcpServerList, err := s.querier.ListMCPServers(ctx)
	if err != nil {
		return nil, err
	}
	mcpServers := make([]schema.MCP, len(mcpServerList))
	for index, mcpServer := range mcpServerList {
		if ms, err := convertDbMCPToSchemaMCP(mcpServer); err == nil {
			mcpServers[index] = ms
		}
	}
	return mcpServers, nil
}

func (s *service) UpdateMCPServer(ctx context.Context, id uuid.UUID, arg schema.MCPUpdate) (schema.MCP, error) {
	updateMCPServerParams := UpdateMCPServerParams{
		Name:                util.GetSQLNullString(arg.Name),
		Endpoint:            util.GetSQLNullString(arg.Endpoint),
		Transport:           *arg.Transport,
		Command:             marshalStringSlice(arg.Command),
		Args:                marshalStringSlice(arg.Args),
		AuthConfig:          util.ToRawMessage[schema.MCPUpdateAuthConfig](&arg.AuthConfig),
		AllowedTools:        marshalStringSlice(arg.AllowedTools),
		RequireConfirmation: util.GetSQLNullBool(arg.RequireConfirmation),
		ConfirmationRules:   util.ToRawMessage[schema.MCPUpdateConfirmationRules](&arg.ConfirmationRules),
		IsActive:            util.GetSQLNullBool(arg.IsActive),
		ID:                  id,
	}

	if arg.AuthType != nil {
		updateMCPServerParams.AuthType = *arg.AuthType
	}

	mcpServer, err := s.querier.UpdateMCPServer(ctx, updateMCPServerParams)
	if err != nil {
		return schema.MCP{}, err
	}
	return convertDbMCPToSchemaMCP(mcpServer)
}

func convertDbMCPToSchemaMCP(mcpServer McpServer) (schema.MCP, error) {
	var authConfig schema.MCPAuthConfig
	if mcpServer.AuthConfig != nil {
		if err := json.Unmarshal(mcpServer.AuthConfig, &authConfig); err != nil {
			return schema.MCP{}, err
		}
	}
	var confirmationRules schema.MCPConfirmationRules
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
		Command:             unmarshalStringSlice(mcpServer.Command),
		ConfirmationRules:   confirmationRules,
		CreatedAt:           mcpServer.CreatedAt,
		Endpoint:            mcpServer.Endpoint,
		ID:                  mcpServer.ID,
		IsActive:            mcpServer.IsActive,
		Name:                mcpServer.Name,
		RequireConfirmation: mcpServer.RequireConfirmation,
		Transport:           mcpServer.Transport,
		UpdatedAt:           mcpServer.UpdatedAt,
		UserID:              mcpServer.UserID,
	}, nil
}

func marshalStringSlice(s []string) string {
	if len(s) == 0 {
		return "[]"
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func unmarshalStringSlice(s string) []string {
	var result []string
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil
	}
	return result
}
