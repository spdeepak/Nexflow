package mcpserver

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	cfg "github.com/spdeepak/nexflow/internal/config"
	"github.com/spdeepak/nexflow/internal/db"
	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/schema"
	"github.com/spdeepak/nexflow/internal/users"
)

type testFixture struct {
	test        *testing.T
	mcpService  Service
	userService users.Service
	conn        *sql.DB
}

func newTestFixture(test *testing.T) *testFixture {
	test.Helper()

	dbConfig := cfg.DBConfig{
		DBName:          "app.db",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,
	}

	require.NoError(test, db.RunMigrations(dbConfig))
	test.Cleanup(func() {
		require.NoError(test, os.Remove("app.db"))
		require.NoError(test, os.Remove("app.db-shm"))
		require.NoError(test, os.Remove("app.db-wal"))
	})

	conn := db.Connect(dbConfig)

	return &testFixture{
		test:        test,
		mcpService:  NewService(conn),
		userService: users.NewService(users.New(conn)),
		conn:        conn,
	}
}

func (tf *testFixture) createUser() schema.User {
	tf.test.Helper()
	user, err := tf.userService.CreateAppUser(context.Background(), uuid.NewString(), "Deepak")
	require.NoError(tf.test, err)
	require.NotNil(tf.test, user)
	return user
}

func (tf *testFixture) createMCPServer() schema.MCP {
	tf.test.Helper()
	user := tf.createUser()
	mcpCreate := schema.MCPCreate{
		Name:                "mcp_name",
		AllowedTools:        []string{"generate_uuid", "shorten_url"},
		Args:                []string{"arg-1", "arg-2"},
		AuthConfig:          nil,
		AuthType:            nil,
		Command:             new("docker"),
		ConfirmationRules:   nil,
		Endpoint:            "https://aisenseapi.com/mcp",
		IsActive:            true,
		RequireConfirmation: false,
		Transport:           enums.McpTransportStreamableHttp,
		UserID:              user.ID,
	}
	createdMCP, err := tf.mcpService.CreateMCPServer(context.Background(), mcpCreate)
	require.NoError(tf.test, err)
	require.NotEmpty(tf.test, createdMCP)
	require.NotEmpty(tf.test, createdMCP.ID)
	require.Equal(tf.test, mcpCreate.Name, createdMCP.Name)
	require.Equal(tf.test, mcpCreate.AllowedTools, createdMCP.AllowedTools)
	require.Equal(tf.test, mcpCreate.Args, createdMCP.Args)
	require.Nil(tf.test, createdMCP.AuthConfig)
	require.Empty(tf.test, createdMCP.AuthType)
	require.Equal(tf.test, mcpCreate.Command, createdMCP.Command)
	require.Empty(tf.test, createdMCP.ConfirmationRules)
	require.Equal(tf.test, mcpCreate.Endpoint, *createdMCP.Endpoint)
	require.Equal(tf.test, mcpCreate.IsActive, createdMCP.IsActive)
	require.Equal(tf.test, mcpCreate.RequireConfirmation, createdMCP.RequireConfirmation)
	require.Equal(tf.test, mcpCreate.Transport, createdMCP.Transport)
	require.Equal(tf.test, mcpCreate.UserID, createdMCP.UserID)
	return createdMCP
}

func TestCreateMCPServer_OK(t *testing.T) {
	fixture := newTestFixture(t)
	fixture.createMCPServer()
}

func TestCreateMCPServer_NOK_EmptyName(t *testing.T) {
	fixture := newTestFixture(t)
	user := fixture.createUser()
	mcpCreate := schema.MCPCreate{
		AllowedTools:        []string{"generate_uuid", "shorten_url"},
		Args:                []string{},
		AuthConfig:          nil,
		AuthType:            nil,
		Command:             new("docker"),
		ConfirmationRules:   nil,
		Endpoint:            "https://aisenseapi.com/mcp",
		IsActive:            true,
		RequireConfirmation: false,
		Transport:           enums.McpTransportStreamableHttp,
		UserID:              user.ID,
	}
	createdMCP, err := fixture.mcpService.CreateMCPServer(context.Background(), mcpCreate)
	require.Error(t, err)
	require.Empty(t, createdMCP)
}

func TestCreateMCPServer_NOK_EmptyEndpoint(t *testing.T) {
	fixture := newTestFixture(t)
	user := fixture.createUser()
	mcpCreate := schema.MCPCreate{
		Name:                "mcp_name",
		AllowedTools:        []string{"generate_uuid", "shorten_url"},
		Args:                []string{},
		AuthConfig:          nil,
		AuthType:            nil,
		Command:             new("docker"),
		ConfirmationRules:   nil,
		IsActive:            true,
		RequireConfirmation: false,
		Transport:           enums.McpTransportStreamableHttp,
		UserID:              user.ID,
	}
	createdMCP, err := fixture.mcpService.CreateMCPServer(context.Background(), mcpCreate)
	require.Error(t, err)
	require.Empty(t, createdMCP)
}

func TestListMCPServers_OK(t *testing.T) {
	fixture := newTestFixture(t)
	fixture.createMCPServer()
	mcpServers, err := fixture.mcpService.ListMCPServers(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, mcpServers)
	require.Len(t, mcpServers, 1)
}

// attachToAgent seeds an agent row and the agent_mcp_server join row directly.
// The mcpserver package cannot import the agents/agentmcp services (they
// already import it), so the fixture writes the relation rows itself.
func (tf *testFixture) attachToAgent(mcpID uuid.UUID) uuid.UUID {
	tf.test.Helper()
	agentID := uuid.New()
	_, err := tf.conn.ExecContext(context.Background(),
		`INSERT INTO agents (id, name) VALUES (?, ?)`, agentID, "agent-"+agentID.String())
	require.NoError(tf.test, err)
	_, err = tf.conn.ExecContext(context.Background(),
		`INSERT INTO agent_mcp_server (agent_id, mcp_server_id) VALUES (?, ?)`, agentID, mcpID)
	require.NoError(tf.test, err)
	return agentID
}

func (tf *testFixture) agentMCPCount(mcpID uuid.UUID) int {
	tf.test.Helper()
	var count int
	err := tf.conn.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM agent_mcp_server WHERE mcp_server_id = ?`, mcpID).Scan(&count)
	require.NoError(tf.test, err)
	return count
}

func TestDeleteMCPServer_OK(t *testing.T) {
	fixture := newTestFixture(t)
	mcp := fixture.createMCPServer()
	fixture.attachToAgent(mcp.ID)
	require.Equal(t, 1, fixture.agentMCPCount(mcp.ID))

	err := fixture.mcpService.DeleteMCPServer(context.Background(), mcp.ID)
	require.NoError(t, err)

	// The join rows must be gone too, not left dangling.
	require.Zero(t, fixture.agentMCPCount(mcp.ID))
	mcpServers, err := fixture.mcpService.ListMCPServers(context.Background())
	require.NoError(t, err)
	require.Empty(t, mcpServers)
}

func TestDeleteMCPServer_Unattached(t *testing.T) {
	fixture := newTestFixture(t)
	mcp := fixture.createMCPServer()

	err := fixture.mcpService.DeleteMCPServer(context.Background(), mcp.ID)
	require.NoError(t, err)
	require.Zero(t, fixture.agentMCPCount(mcp.ID))
}

func TestDeleteMCPServer_NotFound_NamesServer(t *testing.T) {
	fixture := newTestFixture(t)
	missingID := uuid.New()

	err := fixture.mcpService.DeleteMCPServer(context.Background(), missingID)
	require.ErrorContains(t, err, missingID.String())
	require.NotContains(t, err.Error(), "FOREIGN KEY")
}
