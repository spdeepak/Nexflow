package runner

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	adkauth "google.golang.org/adk/v2/auth"

	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/mcpstore"
	"github.com/spdeepak/nexflow/internal/schema"
)

// MCPTransportFor turns a persisted MCP server into a ready-to-connect
// transport. It is the single translation point between schema.MCP and a
// connection, so a run (through mcptoolset) and a one-off tool-discovery call
// (through mcp.Client) connect with byte-identical configuration.
//
// The returned transport already carries its auth round tripper; callers must
// therefore leave mcptoolset.Config.Auth unset.
func MCPTransportFor(ctx context.Context, m schema.MCP, userID uuid.UUID, tokenStore mcpstore.TokenStore) (mcp.Transport, error) {
	provider, err := mcpAuthProvider(ctx, m, userID, tokenStore)
	if err != nil {
		return nil, err
	}

	switch m.Transport {
	case enums.McpTransportStdIO:
		if m.Command == nil || *m.Command == "" {
			return nil, fmt.Errorf("mcp server %q has no command", m.Name)
		}
		if provider != nil {
			// Same rejection mcptoolset applies for a non-HTTP transport with
			// Auth set, but stated in terms the operator can act on.
			return nil, fmt.Errorf("mcp server %q: auth requires the streamable_http transport", m.Name)
		}
		return &mcp.CommandTransport{Command: exec.Command(*m.Command, m.Args...)}, nil
	default:
		if m.Endpoint == nil || *m.Endpoint == "" {
			return nil, fmt.Errorf("mcp server %q has no endpoint", m.Name)
		}
		transport := &mcp.StreamableClientTransport{Endpoint: *m.Endpoint}
		if provider != nil {
			// Mirrors mcptoolset's own (unexported) auth wrapping: a dedicated
			// http.Client whose round tripper injects the credential per request.
			transport.HTTPClient = &http.Client{Transport: &adkauth.Transport{Provider: provider}}
		}
		return transport, nil
	}
}

// mcpAuthProvider resolves the credential for the server's configured auth
// type, or nil when the server declares no usable auth.
func mcpAuthProvider(ctx context.Context, m schema.MCP, userID uuid.UUID, tokenStore mcpstore.TokenStore) (adkauth.CredentialProvider, error) {
	if m.AuthType == nil || m.AuthConfig == nil {
		return nil, nil
	}
	switch *m.AuthType {
	case enums.McpAuthTypeBearerToken:
		if bearer, ok := m.AuthConfig["bearer"].(string); ok {
			return adkauth.StaticToken(bearer), nil
		}
	case enums.McpAuthTypeApiKey:
		name, nameOK := m.AuthConfig["name"].(string)
		value, valueOK := m.AuthConfig["value"].(string)
		if nameOK && valueOK {
			return adkauth.APIKey(name, value), nil
		}
	case enums.McpAuthTypeOAUTH:
		// connectionID == mcpServerID: the grant is keyed on (userID, serverID).
		return mcpstore.CredentialProviderFor(ctx, m.ID.String(), userID.String(), m.ID.String(), tokenStore)
	}
	return nil, nil
}
