package application

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	cfg "github.com/spdeepak/nexflow/internal/config"
	"github.com/spdeepak/nexflow/internal/db"
	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/mcpserver"
	"github.com/spdeepak/nexflow/internal/schema"
	"github.com/spdeepak/nexflow/internal/users"
)

// The frontend never sends userId, so CreateMCP must bind the app user id
// itself before validating, and the persisted row must carry it.
func TestCreateMCP_BindsAppUserID(t *testing.T) {
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
		mcpService: mcpserver.NewService(conn),
	}

	err = app.CreateMCP(schema.MCPCreate{
		Name:      "test_mcp",
		Transport: enums.McpTransportStreamableHttp,
		Endpoint:  "https://example.com/mcp",
		Command:   new(""),
		IsActive:  true,
	})
	require.NoError(t, err)

	mcps, err := app.GetMCPs()
	require.NoError(t, err)
	require.Len(t, mcps, 1)
	require.Equal(t, user.ID, mcps[0].UserID)
}

// An update may omit name entirely: notblank on a nil *string used to reject
// the whole request, which made partial updates impossible. An explicitly blank
// name must still be rejected.
func TestUpdateMCP_PartialUpdateOmitsName(t *testing.T) {
	app, _, mcpID := newTestApp(t, "https://example.com/mcp")

	endpoint := "https://example.com/mcp"
	transport := enums.McpTransport(enums.McpTransportStreamableHttp)

	require.NoError(t, app.UpdateMCP(mcpID.String(), schema.MCPUpdate{
		Transport: &transport,
		Endpoint:  &endpoint,
	}))
	updated, err := app.GetMCP(mcpID.String())
	require.NoError(t, err)
	require.Equal(t, "test-server", updated.Name)

	require.Error(t, app.UpdateMCP(mcpID.String(), schema.MCPUpdate{
		Name:      new(""),
		Transport: &transport,
		Endpoint:  &endpoint,
	}))
}
