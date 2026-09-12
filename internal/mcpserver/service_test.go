package mcpserver

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/spdeepak/nexflow/internal/agentmcp"
	"github.com/spdeepak/nexflow/internal/agents"
	"github.com/spdeepak/nexflow/internal/agentskills"
	cfg "github.com/spdeepak/nexflow/internal/config"
	"github.com/spdeepak/nexflow/internal/db"
	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/modelcredentials"
	"github.com/spdeepak/nexflow/internal/users"
)

type testFixture struct {
	test         *testing.T
	mcpService   Service
	userService  users.Service
	agentService agents.Service
	modelService modelcredentials.Service
}

func newTestFixture(test *testing.T) *testFixture {
	test.Helper()

	dbConfig := cfg.AppConfig{
		DBConfig: cfg.DBConfig{
			Host:              "localhost",
			Port:              "5432",
			DBName:            "app.db",
			UserName:          "admin",
			Password:          "admin",
			SSLMode:           "disable",
			Timeout:           10 * time.Second,
			MaxRetry:          3,
			ConnectTimeout:    5 * time.Second,
			StatementTimeout:  30 * time.Second,
			MaxOpenConns:      1,
			MaxIdleConns:      1,
			ConnMaxLifetime:   1 * time.Hour,
			ConnMaxIdleTime:   30 * time.Minute,
			HealthCheckPeriod: 1 * time.Minute,
		},
	}.DBConfig

	require.NoError(test, db.RunMigrations(dbConfig))
	test.Cleanup(func() {
		require.NoError(test, os.Remove("app.db"))
		require.NoError(test, os.Remove("app.db-shm"))
		require.NoError(test, os.Remove("app.db-wal"))
	})

	conn := db.Connect(dbConfig)

	return &testFixture{
		test:         test,
		mcpService:   NewService(New(conn)),
		userService:  users.NewService(users.New(conn)),
		agentService: agents.NewService(agents.New(conn), agentskills.NewService(agentskills.New(conn)), agentmcp.NewService(agentmcp.New(conn))),
		modelService: modelcredentials.NewService(modelcredentials.New(conn)),
	}
}

func (tf *testFixture) createUser() schema.User {
	tf.test.Helper()
	id := uuid.New()
	user, err := tf.userService.CreateAppUser(context.Background(), id, "Deepak")
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
		Command:             []string{"docker", "command"},
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
	require.Equal(tf.test, mcpCreate.Endpoint, createdMCP.Endpoint)
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
		Command:             []string{"docker", "command"},
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
		Command:             []string{"docker", "command"},
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
