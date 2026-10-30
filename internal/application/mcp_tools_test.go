package application

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// fixtureMCP serves a two-tool MCP server in-process, so tool discovery is
// exercised against a real MCP endpoint without any external dependency.
func fixtureMCP(t *testing.T) *httptest.Server {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{Name: "fixture", Version: "1"}, nil)
	echo := func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{}, nil
	}
	server.AddTool(&mcp.Tool{
		Name:        "generate_uuid",
		Description: "Generates a UUID",
		InputSchema: map[string]any{"type": "object"},
	}, echo)
	server.AddTool(&mcp.Tool{
		Name:        "shorten_url",
		InputSchema: map[string]any{"type": "object"},
	}, echo)

	srv := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil))
	t.Cleanup(srv.Close)
	return srv
}

func TestListMCPTools_ReturnsNamesAndDescriptions(t *testing.T) {
	srv := fixtureMCP(t)
	app, _, mcpID := newTestApp(t, srv.URL)

	tools, err := app.ListMCPTools(mcpID.String())
	require.NoError(t, err)
	require.Len(t, tools, 2)

	names := make([]string, 0, len(tools))
	got := make(map[string]string, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
		got[tool.Name] = tool.Description
	}
	require.ElementsMatch(t, []string{"generate_uuid", "shorten_url"}, names)
	require.Equal(t, "Generates a UUID", got["generate_uuid"])
	require.Equal(t, "", got["shorten_url"])
}

// A server that cannot be reached must surface a clear error rather than an
// empty (and therefore indistinguishable-from-works) tool list.
func TestListMCPTools_UnreachableServerReportsError(t *testing.T) {
	// Port 1 on loopback is never listening: connect fails immediately.
	app, _, mcpID := newTestApp(t, "http://127.0.0.1:1/mcp")

	tools, err := app.ListMCPTools(mcpID.String())
	require.Error(t, err)
	require.ErrorContains(t, err, "connecting to mcp server")
	require.Empty(t, tools)
}
