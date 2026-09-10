package application

import "github.com/spdeepak/nexflow/schema"

func (a *App) CreateMCP(params schema.MCPCreate) error {
	params.UserID = a.deviceID
	_, err := a.mcpService.CreateMCPServer(a.ctx, params)
	return err
}
