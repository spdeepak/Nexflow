package application

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	cfg "github.com/spdeepak/nexflow/internal/config"
	"github.com/spdeepak/nexflow/internal/db"
	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/mcpstore"
	"github.com/spdeepak/nexflow/internal/users"
)

// ConnectOAuthMCP saves grants under the app user UUID and IsOAuthConnected
// must read that same key: a grant stored under any other key (e.g. a legacy
// device-keyed row) has to read as not connected, or the UI would show
// Connected for a server the runner skips at run time.
func TestOAuthGrants_KeyedOnAppUserID(t *testing.T) {
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

	userSvc := users.NewService(users.New(conn))
	user, err := userSvc.CreateAppUser(context.Background(), uuid.NewString(), "Deepak")
	require.NoError(t, err)

	mcpID := uuid.New()
	_, err = conn.ExecContext(context.Background(),
		`INSERT INTO mcp_server (id, user_id, name, endpoint, transport) VALUES (?, ?, ?, ?, ?)`,
		mcpID, user.ID, "oauth-server", "http://localhost:3000/mcp", enums.McpTransportStreamableHttp)
	require.NoError(t, err)

	app := &App{
		ctx:        context.Background(),
		userID:     user.ID,
		tokenStore: mcpstore.NewTokenStore(mcpstore.New(conn)),
	}

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
	other, err := userSvc.CreateAppUser(context.Background(), uuid.NewString(), "Other")
	require.NoError(t, err)
	require.NoError(t, app.tokenStore.DeleteGrant(app.ctx, user.ID.String(), mcpID.String()))
	require.NoError(t, app.tokenStore.SaveGrant(app.ctx, other.ID.String(), mcpID.String(),
		&oauth2.Token{AccessToken: "access-2"}))

	connected, err = app.IsOAuthConnected(mcpID.String())
	require.NoError(t, err)
	require.False(t, connected)
}
