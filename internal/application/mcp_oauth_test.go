package application

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	cfg "github.com/spdeepak/nexflow/internal/config"
	"github.com/spdeepak/nexflow/internal/db"
	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/mcpserver"
	"github.com/spdeepak/nexflow/internal/mcpstore"
	"github.com/spdeepak/nexflow/internal/schema"
	"github.com/spdeepak/nexflow/internal/users"
)

// newTestApp wires a minimal App against a migrated SQLite database and seeds
// one streamable_http MCP server row, so the OAuth and tool-discovery tests
// share the same fixture instead of repeating it.
func newTestApp(t *testing.T, endpoint string) (*App, *sql.DB, uuid.UUID) {
	t.Helper()

	dbConfig := cfg.DBConfig{
		DBName:          "app.db",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,
	}

	require.NoError(t, db.RunMigrations(dbConfig))

	conn := db.Connect(dbConfig)
	t.Cleanup(func() {
		_ = conn.Close()
		_ = os.Remove("app.db")
		_ = os.Remove("app.db-shm")
		_ = os.Remove("app.db-wal")
	})

	user, err := users.NewService(users.New(conn)).CreateAppUser(context.Background(), uuid.NewString(), "Deepak")
	require.NoError(t, err)

	app := &App{
		ctx:        context.Background(),
		userID:     user.ID,
		tokenStore: mcpstore.NewTokenStore(mcpstore.New(conn)),
		mcpService: mcpserver.NewService(conn),
	}
	// Written through the service so the row matches what the app itself stores:
	// command is a plain string in the generated types, so a hand-rolled row with
	// a NULL command would fail to scan back out of ListMCPServers.
	seeded, err := app.mcpService.CreateMCPServer(context.Background(), schema.MCPCreate{
		Name:      "test-server",
		Endpoint:  endpoint,
		Transport: enums.McpTransportStreamableHttp,
		Command:   new(""),
		UserID:    user.ID,
		IsActive:  true,
	})
	require.NoError(t, err)

	return app, conn, seeded.ID
}

// ConnectOAuthMCP saves grants under the app user UUID and IsOAuthConnected
// must read that same key: a grant stored under any other key (e.g. a legacy
// device-keyed row) has to read as not connected, or the UI would show
// Connected for a server the runner skips at run time.
func TestOAuthGrants_KeyedOnAppUserID(t *testing.T) {
	app, conn, mcpID := newTestApp(t, "http://localhost:3000/mcp")

	connected, err := app.IsOAuthConnected(mcpID.String())
	require.NoError(t, err)
	require.False(t, connected)

	// Save/load round trip under the app user id, as ConnectOAuthMCP does.
	require.NoError(t, app.tokenStore.SaveGrant(app.ctx, app.userID.String(), mcpID.String(),
		&oauth2.Token{AccessToken: "access-1"}))
	grant, err := app.tokenStore.LoadGrant(app.ctx, app.userID.String(), mcpID.String())
	require.NoError(t, err)
	require.Equal(t, "access-1", grant.AccessToken)

	connected, err = app.IsOAuthConnected(mcpID.String())
	require.NoError(t, err)
	require.True(t, connected)

	// A grant under a different user must not read as connected: the runner
	// keys on this app user only and would skip the server.
	other, err := users.NewService(users.New(conn)).CreateAppUser(context.Background(), uuid.NewString(), "Other")
	require.NoError(t, err)
	require.NoError(t, app.tokenStore.DeleteGrant(app.ctx, app.userID.String(), mcpID.String()))
	require.NoError(t, app.tokenStore.SaveGrant(app.ctx, other.ID.String(), mcpID.String(),
		&oauth2.Token{AccessToken: "access-2"}))

	connected, err = app.IsOAuthConnected(mcpID.String())
	require.NoError(t, err)
	require.False(t, connected)
}

// Disconnect has to remove the grant under the same (userID, mcpServerID) key
// the other reads use, so IsOAuthConnected flips to false immediately afterwards.
func TestDisconnectOAuthMCP_RemovesGrant(t *testing.T) {
	app, _, mcpID := newTestApp(t, "http://localhost:3000/mcp")

	require.NoError(t, app.tokenStore.SaveGrant(app.ctx, app.userID.String(), mcpID.String(),
		&oauth2.Token{AccessToken: "access-1"}))
	connected, err := app.IsOAuthConnected(mcpID.String())
	require.NoError(t, err)
	require.True(t, connected)

	require.NoError(t, app.DisconnectOAuthMCP(mcpID.String()))

	connected, err = app.IsOAuthConnected(mcpID.String())
	require.NoError(t, err)
	require.False(t, connected)

	// Idempotent: the UI button can be pressed again without an error.
	require.NoError(t, app.DisconnectOAuthMCP(mcpID.String()))
}

func TestDisconnectOAuthMCP_MalformedIDErrors(t *testing.T) {
	app, _, _ := newTestApp(t, "http://localhost:3000/mcp")
	require.Error(t, app.DisconnectOAuthMCP("not-a-uuid"))
}
