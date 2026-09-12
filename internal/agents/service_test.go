package agents

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/agent/llmagent"

	"github.com/spdeepak/nexflow/internal/agentmcp"
	"github.com/spdeepak/nexflow/internal/agentskills"
	cfg "github.com/spdeepak/nexflow/internal/config"
	"github.com/spdeepak/nexflow/internal/db"
	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/mcpserver"
	"github.com/spdeepak/nexflow/internal/modelcredentials"
	"github.com/spdeepak/nexflow/internal/schema"
)

type testFixture struct {
	test              *testing.T
	agentService      Service
	agentSkillService agentskills.Service
	agentMCPService   agentmcp.Service
	mcpService        mcpserver.Service
	userService       users.Service
	modelService      modelcredentials.Service
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
	agentSkillService := agentskills.NewService(agentskills.New(conn))
	agentMCPService := agentmcp.NewService(agentmcp.New(conn))
	mcpService := mcpserver.NewService(mcpserver.New(conn))
	return &testFixture{
		test:              test,
		agentService:      NewService(New(conn), agentSkillService, agentMCPService),
		agentSkillService: agentSkillService,
		agentMCPService:   agentMCPService,
		mcpService:        mcpService,
		userService:       users.NewService(users.New(conn)),
		modelService:      modelcredentials.NewService(modelcredentials.New(conn)),
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

func (tf *testFixture) createUserModel(user schema.User) schema.ModelCredential {
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
	model, err := tf.modelService.CreateUserModelCredential(context.Background(), user.ID, arg)
	require.NoError(tf.test, err)
	require.NotNil(tf.test, model)
	return model
}

func (tf *testFixture) createRootAgent(model schema.ModelCredential) schema.Agent {
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

func (tf *testFixture) createSubAgent(model schema.ModelCredential, rootAgent schema.Agent) schema.Agent {
	return tf.createNamedSubAgent(model, rootAgent, "Test sub-agent")
}

func (tf *testFixture) createNamedSubAgent(model schema.ModelCredential, rootAgent schema.Agent, name string) schema.Agent {
	tf.test.Helper()
	agent, err := tf.agentService.CreateSubAgent(context.Background(), schema.AgentCreate{
		Name:              name,
		Description:       "Test sub-agent description",
		Instruction:       new("Test sub-agent instruction"),
		GlobalInstruction: new("Test sub-agent global instruction"),
		Mode:              llmagent.ModeTask,
		ModelName:         "minimax-m3:cloud",
		ModelCredentialID: model.ID,
		CredentialSource:  enums.CredentialSourceAuto,
		ParentAgentID:     &rootAgent.ID,
	})
	require.NoError(tf.test, err)
	require.NotNil(tf.test, agent)
	return agent
}

func (tf *testFixture) getAgent(id uuid.UUID) schema.Agent {
	tf.test.Helper()
	agent, err := tf.agentService.GetAgent(context.Background(), id)
	require.NoError(tf.test, err)
	require.NotNil(tf.test, agent)
	assert.Equal(tf.test, id, agent.ID)
	return agent
}

func (tf *testFixture) listRootAgents() []schema.Agent {
	tf.test.Helper()
	agents, err := tf.agentService.ListRootAgents(context.Background())
	require.NoError(tf.test, err)
	require.NotNil(tf.test, agents)
	require.NotEmpty(tf.test, agents)
	assert.Len(tf.test, agents, 1)
	return agents
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

func (tf *testFixture) LinkMCP(agentId, mcpId uuid.UUID) {
	tf.test.Helper()
	err := tf.agentMCPService.LinkAgentAndMCP(context.Background(), agentmcp.LinkAgentAndMCPParams{AgentID: agentId, McpServerID: mcpId})
	require.NoError(tf.test, err)
	agentMCPs, err := tf.agentMCPService.ListAgentMCP(context.Background(), agentId)
	require.NoError(tf.test, err)
	require.NotEmpty(tf.test, agentMCPs)
	require.Len(tf.test, agentMCPs, 1)
}

func TestCreateRootAgentWithUserModel_OK(t *testing.T) {
	fixture := newTestFixture(t)
	user := fixture.createUser()
	model := fixture.createUserModel(user)
	fixture.createRootAgent(model)
}

func TestCreateSubAgentWithUserModel_OK(t *testing.T) {
	fixture := newTestFixture(t)
	user := fixture.createUser()
	model := fixture.createUserModel(user)
	rootAgent := fixture.createRootAgent(model)
	fixture.createSubAgent(model, rootAgent)
}

func TestCreateRootAgentWithAppModel_OK(t *testing.T) {
	fixture := newTestFixture(t)
	fixture.createUser()
	model := fixture.createAppModel()
	rootAgent := fixture.createRootAgent(model)
	fixture.createSubAgent(model, rootAgent)
}

func TestGetAgent_OK(t *testing.T) {
	fixture := newTestFixture(t)
	fixture.createUser()
	model := fixture.createAppModel()
	rootAgent := fixture.createRootAgent(model)
	fixture.getAgent(rootAgent.ID)
}

func TestListRootAgents_OK(t *testing.T) {
	fixture := newTestFixture(t)
	fixture.createUser()
	model := fixture.createAppModel()
	fixture.createRootAgent(model)
	fixture.listRootAgents()
}

func TestCreateRootAgentWithAppModel_NOK_NoParentID(t *testing.T) {
	fixture := newTestFixture(t)
	fixture.createUser()
	model := fixture.createAppModel()

	subAgent := schema.AgentCreate{
		Name:              "Test sub-agent",
		Description:       "Test sub-agent description",
		Instruction:       new("Test sub-agent instruction"),
		GlobalInstruction: new("Test sub-agent global instruction"),
		Mode:              llmagent.ModeTask,
		ModelName:         "minimax-m3:cloud",
		ModelCredentialID: model.ID,
		CredentialSource:  enums.CredentialSourceAuto,
	}
	agent, err := fixture.agentService.CreateSubAgent(context.Background(), subAgent)
	assert.Error(t, err)
	assert.Empty(t, agent)
}

func TestSubAgentPositionsAppendInOrder(t *testing.T) {
	fixture := newTestFixture(t)
	fixture.createUser()
	model := fixture.createAppModel()
	rootAgent := fixture.createRootAgent(model)
	sa1 := fixture.createNamedSubAgent(model, rootAgent, "sa1")
	sa2 := fixture.createNamedSubAgent(model, rootAgent, "sa2")
	sa3 := fixture.createNamedSubAgent(model, rootAgent, "sa3")

	children, err := fixture.agentService.GetSubAgents(context.Background(), rootAgent.ID)
	require.NoError(t, err)
	require.Len(t, children, 3)

	assert.Equal(t, sa1.ID, children[0].ID)
	assert.EqualValues(t, 0, *children[0].Position)
	assert.Equal(t, sa2.ID, children[1].ID)
	assert.EqualValues(t, 1, *children[1].Position)
	assert.Equal(t, sa3.ID, children[2].ID)
	assert.EqualValues(t, 2, *children[2].Position)
}

func TestReorderAgentChildren_OK(t *testing.T) {
	fixture := newTestFixture(t)
	fixture.createUser()
	model := fixture.createAppModel()
	rootAgent := fixture.createRootAgent(model)
	sa1 := fixture.createNamedSubAgent(model, rootAgent, "sa1")
	sa2 := fixture.createNamedSubAgent(model, rootAgent, "sa2")
	sa3 := fixture.createNamedSubAgent(model, rootAgent, "sa3")

	err := fixture.agentService.ReorderAgentChildren(
		context.Background(),
		rootAgent.ID,
		[]uuid.UUID{sa3.ID, sa2.ID, sa1.ID},
	)
	require.NoError(t, err)

	children, err := fixture.agentService.GetSubAgents(context.Background(), rootAgent.ID)
	require.NoError(t, err)
	require.Len(t, children, 3)

	assert.Equal(t, sa3.ID, children[0].ID)
	assert.EqualValues(t, 1, *children[0].Position)
	assert.Equal(t, sa2.ID, children[1].ID)
	assert.EqualValues(t, 2, *children[1].Position)
	assert.Equal(t, sa1.ID, children[2].ID)
	assert.EqualValues(t, 3, *children[2].Position)
}

func TestUpdateAgent_OKWithoutForeignKeys(t *testing.T) {
	fixture := newTestFixture(t)
	fixture.createUser()
	model := fixture.createAppModel()
	rootAgent := fixture.createRootAgent(model)

	credSource := enums.CredentialSourceApp
	desc := "new description"
	instruction := "new instruction"
	globalInstruction := "new global instruction"
	isActive := false
	mode := llmagent.ModeTask
	modelName := "gemma4:cloud"
	agentUpdate := schema.AgentUpdate{
		CredentialSource:  &credSource,
		Description:       &desc,
		GlobalInstruction: &globalInstruction,
		Instruction:       &instruction,
		IsActive:          &isActive,
		Mode:              &mode,
		ModelName:         &modelName,
	}
	agent, err := fixture.agentService.UpdateAgent(context.Background(), rootAgent.ID, agentUpdate)
	require.NoError(t, err)
	require.Equal(t, desc, agent.Description.String)
	require.Equal(t, credSource, agent.CredentialSource)
	require.Equal(t, rootAgent.ID, agent.ID)
	require.Equal(t, rootAgent.Name, agent.Name)
	require.Empty(t, rootAgent.ParentAgentID)
	require.Empty(t, agent.ParentAgentID)
	require.Equal(t, instruction, agent.Instruction.String)
	require.Equal(t, globalInstruction, agent.GlobalInstruction.String)
	require.Equal(t, mode, agent.Mode)
	require.Equal(t, modelName, agent.ModelName.String)
	require.Equal(t, rootAgent.ModelCredentialID, agent.ModelCredentialID)
	require.Equal(t, rootAgent.ModelConfig, agent.ModelConfig)
	require.Equal(t, rootAgent.ConfigJson, agent.ConfigJson)
	require.Equal(t, isActive, agent.IsActive)
	require.Equal(t, rootAgent.Position, agent.Position)
	require.Equal(t, rootAgent.CreatedAt, agent.CreatedAt)
	require.Equal(t, rootAgent.UpdatedAt, agent.UpdatedAt)
}

func TestUpdateAgent_OKWithForeignKeys(t *testing.T) {
	fixture := newTestFixture(t)
	user := fixture.createUser()
	model := fixture.createAppModel()
	newModel := fixture.createUserModel(user)
	rootAgent := fixture.createRootAgent(model)

	agentUpdate := schema.AgentUpdate{
		ModelCredentialID: &newModel.ID,
	}
	agent, err := fixture.agentService.UpdateAgent(context.Background(), rootAgent.ID, agentUpdate)
	require.NoError(t, err)
	require.Equal(t, rootAgent.Description.String, agent.Description.String)
	require.Equal(t, rootAgent.CredentialSource, agent.CredentialSource)
	require.Equal(t, rootAgent.ID, agent.ID)
	require.Equal(t, rootAgent.Name, agent.Name)
	require.Empty(t, rootAgent.ParentAgentID)
	require.Empty(t, agent.ParentAgentID)
	require.Equal(t, agent.Instruction.String, agent.Instruction.String)
	require.Equal(t, agent.GlobalInstruction.String, agent.GlobalInstruction.String)
	require.Equal(t, rootAgent.Mode, agent.Mode)
	require.Equal(t, newModel.ModelName, agent.ModelName.String)
	require.Equal(t, newModel.ID.String(), agent.ModelCredentialID.String())
	require.Equal(t, rootAgent.ModelConfig, agent.ModelConfig)
	require.Equal(t, rootAgent.ConfigJson, agent.ConfigJson)
	require.Equal(t, rootAgent.IsActive, agent.IsActive)
	require.Equal(t, rootAgent.Position, agent.Position)
	require.Equal(t, rootAgent.CreatedAt, agent.CreatedAt)
	require.Equal(t, rootAgent.UpdatedAt, agent.UpdatedAt)
}

func TestDeleteAgent_OK(t *testing.T) {
	fixture := newTestFixture(t)
	model := fixture.createAppModel()
	rootAgent := fixture.createRootAgent(model)
	err := fixture.agentService.DeleteAgent(context.Background(), rootAgent.ID)
	require.NoError(t, err)
	_, err = fixture.agentService.GetAgent(context.Background(), rootAgent.ID)
	require.Error(t, err)
}

func TestGetAgentDetail_OK(t *testing.T) {
	fixture := newTestFixture(t)
	model := fixture.createAppModel()
	rootAgent := fixture.createRootAgent(model)
	mcp := fixture.createMCPServer()
	fixture.LinkMCP(rootAgent.ID, mcp.ID)

	agentDetail, err := fixture.agentService.GetAgentDetail(context.Background(), rootAgent.ID)
	require.NoError(t, err)
	require.NotEmpty(t, agentDetail)

	require.Equal(t, rootAgent.Description.String, agentDetail.Description)
	require.Equal(t, rootAgent.CredentialSource, agentDetail.CredentialSource)
	require.Equal(t, rootAgent.ID, agentDetail.ID)
	require.Equal(t, rootAgent.Name, agentDetail.Name)
	require.Empty(t, rootAgent.ParentAgentID)
	require.Empty(t, agentDetail.ParentAgentID)
	require.Equal(t, rootAgent.Instruction.String, agentDetail.Instruction)
	require.Equal(t, rootAgent.GlobalInstruction.String, *agentDetail.GlobalInstruction)
	require.Equal(t, rootAgent.Mode, agentDetail.Mode)
	require.Equal(t, rootAgent.ModelName.String, agentDetail.ModelName)
	require.Equal(t, model.ID.String(), agentDetail.ModelCredentialID.String())
	require.Nil(t, rootAgent.ModelConfig)
	require.Nil(t, agentDetail.ModelConfig)
	require.Nil(t, rootAgent.ConfigJson)
	require.Nil(t, agentDetail.ConfigJson)
	require.Equal(t, rootAgent.IsActive, agentDetail.IsActive)
	require.Zero(t, rootAgent.Position)
	require.Nil(t, agentDetail.Position)
	require.Equal(t, rootAgent.CreatedAt, agentDetail.CreatedAt)
	require.Equal(t, rootAgent.UpdatedAt, agentDetail.UpdatedAt)
}

func TestGetAgentDetail_NOK_AgentNotFound(t *testing.T) {
	fixture := newTestFixture(t)

	agentDetail, err := fixture.agentService.GetAgentDetail(context.Background(), uuid.New())
	require.Error(t, err)
	require.Empty(t, agentDetail)
}

func TestGetRootAgent_OK(t *testing.T) {
	fixture := newTestFixture(t)
	model := fixture.createAppModel()
	rootAgent := fixture.createRootAgent(model)

	agent, err := fixture.agentService.GetRootAgent(context.Background(), rootAgent.ID)
	require.NoError(t, err)
	require.NotEmpty(t, agent)
	require.Equal(t, rootAgent.CreatedAt, agent.CreatedAt)
	require.Equal(t, rootAgent.CredentialSource, agent.CredentialSource)
	require.Equal(t, rootAgent.Description.String, agent.Description)
	require.Equal(t, rootAgent.GlobalInstruction.String, *agent.GlobalInstruction)
	require.Equal(t, rootAgent.ID, agent.ID)
	require.Equal(t, rootAgent.Instruction.String, agent.Instruction)
	require.Equal(t, rootAgent.IsActive, agent.IsActive)
	require.Equal(t, rootAgent.Mode, agent.Mode)
	require.Equal(t, rootAgent.ModelCredentialID, agent.ModelCredentialID)
	require.Equal(t, rootAgent.ModelName.String, agent.ModelName)
	require.Equal(t, rootAgent.Name, agent.Name)
	require.Equal(t, rootAgent.UpdatedAt, agent.UpdatedAt)
}

func TestGetRootAgent_NOK_GetError(t *testing.T) {
	fixture := newTestFixture(t)
	agent, err := fixture.agentService.GetRootAgent(context.Background(), uuid.New())
	require.Error(t, err)
	require.Empty(t, agent)
}
