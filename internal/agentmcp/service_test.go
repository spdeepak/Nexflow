package agentmcp_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/agent/llmagent"

	"github.com/spdeepak/nexflow/internal/agentmcp"
	"github.com/spdeepak/nexflow/internal/agents"
	"github.com/spdeepak/nexflow/internal/agentskills"
	cfg "github.com/spdeepak/nexflow/internal/config"
	"github.com/spdeepak/nexflow/internal/db"
	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/mcpserver"
	"github.com/spdeepak/nexflow/internal/modelcredentials"
	"github.com/spdeepak/nexflow/internal/users"
	"github.com/spdeepak/nexflow/schema"
)

type testFixture struct {
	test            *testing.T
	agentMcpService agentmcp.Service
	mcpService      mcpserver.Service
	userService     users.Service
	agentService    agents.Service
	modelService    modelcredentials.Service
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
		test:            test,
		agentMcpService: agentmcp.NewService(agentmcp.New(conn)),
		mcpService:      mcpserver.NewService(mcpserver.New(conn)),
		userService:     users.NewService(users.New(conn)),
		agentService:    agents.NewService(agents.New(conn), agentskills.NewService(agentskills.New(conn)), agentmcp.NewService(agentmcp.New(conn))),
		modelService:    modelcredentials.NewService(modelcredentials.New(conn)),
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

func (tf *testFixture) createAppModel() schema.ModelCredential {
	tf.test.Helper()
	arg := schema.ModelCredentialCreate{
		ApiKey:      "schema-key",
		BaseUrl:     "http://localhost:11434/v1",
		ExtraConfig: schema.ModelCredentialCreateExtraConfig{},
		Provider:    "ollama",
		ModelName:   "minimax-m3:cloud",
		Scope:       enums.CredentialScopeApp,
		Title:       "test ollama minimax",
	}
	model, err := tf.modelService.CreateAppModelCredential(context.Background(), arg)
	require.NoError(tf.test, err)
	require.NotNil(tf.test, model)
	return model
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

func (tf *testFixture) createRootAgent(model schema.ModelCredential) agents.Agent {
	tf.test.Helper()
	agent, err := tf.agentService.CreateRootAgent(context.Background(), schema.AgentCreate{
		Name:              "Test agent",
		Description:       "Test agent description",
		Instruction:       new("Test agent instruction"),
		GlobalInstruction: new("Test agent global instruction"),
		Mode:              llmagent.ModeChat,
		ModelName:         "minimax-m3:cloud",
		ModelCredentialID: model.ID,
		CredentialSource:  enums.CredentialSourceAuto,
	})
	require.NoError(tf.test, err)
	require.NotEmpty(tf.test, agent)
	require.Equal(tf.test, "Test agent", agent.Name)
	require.Equal(tf.test, "Test agent description", agent.Description.String)
	require.Equal(tf.test, "Test agent instruction", agent.Instruction.String)
	require.Equal(tf.test, "Test agent global instruction", agent.GlobalInstruction.String)
	require.Equal(tf.test, llmagent.ModeChat, agent.Mode)
	require.Equal(tf.test, "minimax-m3:cloud", agent.ModelName.String)
	require.Equal(tf.test, model.ID, agent.ModelCredentialID)
	require.Equal(tf.test, enums.CredentialSourceAuto, agent.CredentialSource)
	return agent
}

func TestLinkAgentAndMCP_OK(t *testing.T) {
	fixture := newTestFixture(t)
	model := fixture.createAppModel()
	agent := fixture.createRootAgent(model)
	mcp := fixture.createMCPServer()

	err := fixture.agentMcpService.LinkAgentAndMCP(context.Background(), agentmcp.LinkAgentAndMCPParams{AgentID: agent.ID, McpServerID: mcp.ID})
	require.NoError(t, err)
	agentMCPs, err := fixture.agentMcpService.ListAgentMCP(context.Background(), agent.ID)
	require.NoError(t, err)
	require.NotEmpty(t, agentMCPs)
	require.Len(t, agentMCPs, 1)
}

func TestDetachMCPFromAgent_OK(t *testing.T) {
	fixture := newTestFixture(t)
	model := fixture.createAppModel()
	agent := fixture.createRootAgent(model)
	mcp := fixture.createMCPServer()

	err := fixture.agentMcpService.LinkAgentAndMCP(context.Background(), agentmcp.LinkAgentAndMCPParams{AgentID: agent.ID, McpServerID: mcp.ID})
	require.NoError(t, err)
	agentMCPs, err := fixture.agentMcpService.ListAgentMCP(context.Background(), agent.ID)
	require.NoError(t, err)
	require.NotEmpty(t, agentMCPs)
	require.Len(t, agentMCPs, 1)
	err = fixture.agentMcpService.DetachMCPFromAgent(context.Background(), agentmcp.DetachMCPFromAgentParams{AgentID: agent.ID, McpServerID: mcp.ID})
	require.NoError(t, err)
	agentMCPs, err = fixture.agentMcpService.ListAgentMCP(context.Background(), agent.ID)
	require.NoError(t, err)
	require.Empty(t, agentMCPs)
}
