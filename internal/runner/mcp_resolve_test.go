package runner

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/mcpstore"
	"github.com/spdeepak/nexflow/internal/schema"
)

// noGrantStore simulates an OAuth MCP server whose client config exists but
// whose grant is missing — CredentialProviderFor fails with sql.ErrNoRows.
type noGrantStore struct{ mcpstore.TokenStore }

func (noGrantStore) ClientConfig(context.Context, string) (mcpstore.OAuthClientConfig, error) {
	return mcpstore.OAuthClientConfig{
		AuthURL:  "https://auth.example.com/authorize",
		TokenURL: "https://auth.example.com/token",
		ClientID: "client-1",
	}, nil
}

func (noGrantStore) LoadGrant(context.Context, string, string) (mcpstore.OAuthGrant, error) {
	return mcpstore.OAuthGrant{}, sql.ErrNoRows
}

// A configured-but-unauthenticated OAuth server is skipped with a warning (the
// run must not fail), and the skip must drop it entirely rather than leave a
// nil toolset in the slice handed to llmagent.Config.Toolsets.
func TestResolveMCPs_OAuthWithoutGrantLeavesNoNil(t *testing.T) {
	r := &runner{userID: uuid.New(), tokenStore: noGrantStore{}}
	endpoint := "https://example.com/mcp"
	oauthType := enums.McpAuthType(enums.McpAuthTypeOAUTH)

	agent := schema.AgentDetail{Mcps: []schema.MCP{
		{
			ID:         uuid.New(),
			Name:       "oauth-no-grant",
			Transport:  enums.McpTransportStreamableHttp,
			Endpoint:   &endpoint,
			AuthType:   &oauthType,
			AuthConfig: schema.AuthConfig{"client_id": "client-1"},
		},
		{
			ID:        uuid.New(),
			Name:      "plain",
			Transport: enums.McpTransportStreamableHttp,
			Endpoint:  &endpoint,
		},
	}}

	got := r.resolveMCPs(context.Background(), agent)
	require.Len(t, got, 1, "the unauthenticated oauth server must be dropped, not kept as a nil entry")
	for i, ts := range got {
		require.NotNil(t, ts, "toolset at index %d must not be nil", i)
	}
}
