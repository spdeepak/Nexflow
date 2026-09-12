package application

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/schema"
)

func (a *App) CreateMCP(params schema.MCPCreate) error {
	if err := params.Validate(); err != nil {
		return err
	}
	params.UserID = a.deviceID
	_, err := a.mcpService.CreateMCPServer(a.ctx, params)
	return err
}

func (a *App) GetMCPs() ([]schema.MCP, error) {
	return a.mcpService.ListMCPServers(a.ctx)
}

func (a *App) GetMCP(mcpId string) (schema.MCP, error) {
	id, err := uuid.Parse(mcpId)
	if err != nil {
		return schema.MCP{}, fmt.Errorf("invalid mcp id: %s", mcpId)
	}
	mcps, err := a.mcpService.ListMCPServers(a.ctx)
	if err != nil {
		return schema.MCP{}, err
	}
	for _, m := range mcps {
		if m.ID == id {
			return m, nil
		}
	}
	return schema.MCP{}, fmt.Errorf("mcp not found: %s", mcpId)
}

func (a *App) UpdateMCP(mcpId string, params schema.MCPUpdate) error {
	if err := params.Validate(); err != nil {
		return err
	}
	id, err := uuid.Parse(mcpId)
	if err != nil {
		return fmt.Errorf("invalid mcp id: %s", mcpId)
	}
	_, err = a.mcpService.UpdateMCPServer(a.ctx, id, params)
	return err
}

func (a *App) DeleteMCP(mcpId string) error {
	id, err := uuid.Parse(mcpId)
	if err != nil {
		return fmt.Errorf("invalid mcp id: %s", mcpId)
	}
	return a.mcpService.DeleteMCPServer(a.ctx, id)
}
