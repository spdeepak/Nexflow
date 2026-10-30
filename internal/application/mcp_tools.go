package application

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/spdeepak/nexflow/internal/runner"
	"github.com/spdeepak/nexflow/internal/schema"
)

// mcpListToolsTimeout bounds a discovery call so an unreachable server reports
// an error instead of leaving the UI waiting forever.
const mcpListToolsTimeout = 30 * time.Second

// ListMCPTools connects to the MCP server using exactly the configuration a run
// uses and returns the tools it advertises. Discovery is on-demand only: nothing
// is cached, the stored allow-list is never read, and no caller here writes it.
func (a *App) ListMCPTools(mcpID string) ([]schema.MCPTool, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	mcpServer, err := a.GetMCP(mcpID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, mcpListToolsTimeout)
	defer cancel()

	transport, err := runner.MCPTransportFor(ctx, mcpServer, a.userID, a.tokenStore)
	if err != nil {
		return nil, err
	}

	// Deliberately not mcptoolset: it keeps its session with no exported Close,
	// so a stdio server spawned for discovery would outlive the call. Opening a
	// session here lets defer session.Close() reap it.
	client := mcp.NewClient(&mcp.Implementation{Name: "nexflow", Version: "1"}, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connecting to mcp server %q: %w", mcpServer.Name, err)
	}
	defer func() { _ = session.Close() }()

	var tools []schema.MCPTool
	cursor := ""
	for {
		res, err := session.ListTools(ctx, &mcp.ListToolsParams{Cursor: cursor})
		if err != nil {
			return nil, fmt.Errorf("listing tools from mcp server %q: %w", mcpServer.Name, err)
		}
		for _, t := range res.Tools {
			tools = append(tools, schema.MCPTool{Name: t.Name, Description: t.Description})
		}
		if res.NextCursor == "" {
			return tools, nil
		}
		cursor = res.NextCursor
	}
}
