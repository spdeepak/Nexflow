package agentskills_test

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
	"github.com/spdeepak/nexflow/internal/modelcredentials"
	"github.com/spdeepak/nexflow/internal/schema"
	"github.com/spdeepak/nexflow/internal/skills"
)

type testFixture struct {
	test              *testing.T
	agentSkillService agentskills.Service
	skillService      skills.Service
	userService       users.Service
	agentService      agents.Service
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

	return &testFixture{
		test:              test,
		agentSkillService: agentskills.NewService(agentskills.New(conn)),
		userService:       users.NewService(users.New(conn)),
		skillService:      skills.NewService(skills.New(conn)),
		agentService:      agents.NewService(agents.New(conn), agentskills.NewService(agentskills.New(conn)), agentmcp.NewService(agentmcp.New(conn))),
		modelService:      modelcredentials.NewService(modelcredentials.New(conn)),
	}
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

func (tf *testFixture) createUser() schema.User {
	tf.test.Helper()
	id := uuid.New()
	user, err := tf.userService.CreateAppUser(context.Background(), id, "Deepak")
	require.NoError(tf.test, err)
	require.NotNil(tf.test, user)
	return user
}

func (tf *testFixture) createSkill(userId uuid.UUID) schema.Skill {
	tf.test.Helper()
	content := "skill content"
	arg := schema.SkillCreate{
		Content:     &content,
		ContentType: enums.SkillContentTypeText,
		Scope:       enums.SkillScopeUser,
		Title:       "test skill one",
	}
	skill, err := tf.skillService.CreateSkill(context.Background(), arg, userId)
	require.NoError(tf.test, err)
	require.NotNil(tf.test, skill)
	require.Equal(tf.test, *arg.Content, *skill.Content)
	require.Equal(tf.test, arg.ContentType, skill.ContentType)
	require.Equal(tf.test, arg.Scope, skill.Scope)
	require.Equal(tf.test, arg.Title, skill.Title)
	return skill
}

func TestAttachSkillToAgent_OK(t *testing.T) {
	fixture := newTestFixture(t)
	model := fixture.createAppModel()
	agent := fixture.createRootAgent(model)
	user := fixture.createUser()
	skill := fixture.createSkill(user.ID)

	err := fixture.agentSkillService.AttachSkillToAgent(context.Background(), agentskills.AttachSkillToAgentParams{AgentID: agent.ID, SkillID: skill.ID})
	require.NoError(t, err)

	agentSkill, err := fixture.agentSkillService.ListAgentSkill(context.Background(), agent.ID)
	require.NoError(t, err)
	require.NotEmpty(t, agentSkill)

	require.Equal(t, *agentSkill[0].Content, *skill.Content)
	require.Equal(t, agentSkill[0].ContentType, skill.ContentType)
	require.Equal(t, agentSkill[0].Scope, skill.Scope)
	require.Equal(t, agentSkill[0].Title, skill.Title)
}

func TestDetachSkillToAgent_OK(t *testing.T) {
	fixture := newTestFixture(t)
	model := fixture.createAppModel()
	agent := fixture.createRootAgent(model)
	user := fixture.createUser()
	skill := fixture.createSkill(user.ID)

	err := fixture.agentSkillService.AttachSkillToAgent(context.Background(), agentskills.AttachSkillToAgentParams{AgentID: agent.ID, SkillID: skill.ID})
	require.NoError(t, err)

	agentSkill, err := fixture.agentSkillService.ListAgentSkill(context.Background(), agent.ID)
	require.NoError(t, err)
	require.NotEmpty(t, agentSkill)

	require.Equal(t, *agentSkill[0].Content, *skill.Content)
	require.Equal(t, agentSkill[0].ContentType, skill.ContentType)
	require.Equal(t, agentSkill[0].Scope, skill.Scope)
	require.Equal(t, agentSkill[0].Title, skill.Title)

	err = fixture.agentSkillService.DetachSkillFromAgent(context.Background(), agentskills.DetachSkillFromAgentParams{AgentID: agent.ID, SkillID: skill.ID})
	require.NoError(t, err)

	agentSkill, err = fixture.agentSkillService.ListAgentSkill(context.Background(), agent.ID)
	require.NoError(t, err)
	require.Empty(t, agentSkill)
}
