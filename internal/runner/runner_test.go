//go:build integration

package runner

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/session"

	"github.com/spdeepak/nexflow/internal/agentmcp"
	"github.com/spdeepak/nexflow/internal/agents"
	"github.com/spdeepak/nexflow/internal/agentskills"
	cfg "github.com/spdeepak/nexflow/internal/config"
	"github.com/spdeepak/nexflow/internal/db"
	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/events"
	"github.com/spdeepak/nexflow/internal/modelcredentials"
	"github.com/spdeepak/nexflow/internal/runs"
	"github.com/spdeepak/nexflow/internal/schema"
	"github.com/spdeepak/nexflow/internal/sessions"
	"github.com/spdeepak/nexflow/internal/sessionstate"
)

type testFixture struct {
	test              *testing.T
	agentService      agents.Service
	userService       users.Service
	modelService      modelcredentials.Service
	sessionService    session.Service
	sessionsQuery     sessions.Querier
	eventsQuery       events.Querier
	sessionStateQuery sessionstate.Querier
	runsQuery         runs.Querier
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

	sessionsQuery := sessions.New(conn)
	eventsQuery := events.New(conn)
	sessionStateQuery := sessionstate.New(conn)
	runsQuery := runs.New(conn)

	return &testFixture{
		test:              test,
		agentService:      agents.NewService(agents.New(conn), agentskills.NewService(agentskills.New(conn)), agentmcp.NewService(agentmcp.New(conn))),
		userService:       users.NewService(users.New(conn)),
		modelService:      modelcredentials.NewService(modelcredentials.New(conn)),
		sessionService:    sessions.NewSessionStore(sessionsQuery, eventsQuery, sessionStateQuery, "nexflow"),
		sessionsQuery:     sessionsQuery,
		eventsQuery:       eventsQuery,
		sessionStateQuery: sessionStateQuery,
		runsQuery:         runsQuery,
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
		ModelName:   "gemma4:31b-cloud",
		Scope:       enums.CredentialScopeApp,
		Title:       "ollama gemma4 cloud",
	}
	model, err := tf.modelService.CreateAppModelCredential(context.Background(), arg)
	require.NoError(tf.test, err)
	require.NotNil(tf.test, model)
	return model
}

func (tf *testFixture) createResearcherSubAgent(model schema.ModelCredential, rootAgentId uuid.UUID) schema.Agent {
	tf.test.Helper()
	agentCreate := schema.AgentCreate{
		Name:        "researcher",
		Description: "Researches and explains technical topics.",
		Instruction: new(`
					You are a senior software engineer acting as a researcher.
					
					Your job is to analyze the user's question and produce a technically
					accurate research report.
					
					Focus on:
					- architecture
					- important concepts
					- advantages
					- disadvantages
					- practical production considerations
					
					Do not invent facts.
					
					Produce a concise but technically detailed report.
					`),
		Mode:              llmagent.ModeSingleTurn,
		ModelName:         "gemma4:31b-cloud",
		ModelCredentialID: model.ID,
		CredentialSource:  enums.CredentialSourceAuto,
		ParentAgentID:     &rootAgentId,
	}
	agent, err := tf.agentService.CreateSubAgent(context.Background(), agentCreate)
	require.NoError(tf.test, err)
	require.NotEmpty(tf.test, agent)
	require.Equal(tf.test, agentCreate.Name, agent.Name)
	require.Equal(tf.test, agentCreate.Description, agent.Description.String)
	require.Equal(tf.test, *agentCreate.Instruction, agent.Instruction.String)
	require.False(tf.test, agent.GlobalInstruction.Valid)
	require.Equal(tf.test, llmagent.ModeSingleTurn, agent.Mode)
	require.Equal(tf.test, agentCreate.ModelName, agent.ModelName.String)
	require.Equal(tf.test, model.ID, agent.ModelCredentialID)
	require.Equal(tf.test, enums.CredentialSourceAuto, agent.CredentialSource)
	return agent
}

func (tf *testFixture) createReviewerSubAgent(model schema.ModelCredential, rootAgentId uuid.UUID) schema.Agent {
	tf.test.Helper()
	agentCreate := schema.AgentCreate{
		Name:        "reviewer",
		Description: "Reviews technical research for correctness.",
		Instruction: new(`
					You are a senior software engineer reviewing another engineer's
					technical research.
					
					Look for:
					- factual errors
					- unsupported claims
					- missing trade-offs
					- architectural problems
					- misleading statements
					
					Then produce a corrected final answer.
					
					Do not mention that you are reviewing another agent.
					Return only the final technical answer along with appending the current time in Germany at the end.
					`),
		Mode:              llmagent.ModeSingleTurn,
		ModelName:         "gemma4:31b-cloud",
		ModelCredentialID: model.ID,
		CredentialSource:  enums.CredentialSourceAuto,
		ParentAgentID:     &rootAgentId,
	}
	agent, err := tf.agentService.CreateSubAgent(context.Background(), agentCreate)
	require.NoError(tf.test, err)
	require.NotEmpty(tf.test, agent)
	require.Equal(tf.test, agentCreate.Name, agent.Name)
	require.Equal(tf.test, agentCreate.Description, agent.Description.String)
	require.Equal(tf.test, *agentCreate.Instruction, agent.Instruction.String)
	require.False(tf.test, agent.GlobalInstruction.Valid)
	require.Equal(tf.test, llmagent.ModeSingleTurn, agent.Mode)
	require.Equal(tf.test, agentCreate.ModelName, agent.ModelName.String)
	require.Equal(tf.test, model.ID, agent.ModelCredentialID)
	require.Equal(tf.test, enums.CredentialSourceAuto, agent.CredentialSource)
	return agent
}

func (tf *testFixture) createTechnicalResearchTeamRootAgent(model schema.ModelCredential) schema.Agent {
	tf.test.Helper()
	agentCreate := schema.AgentCreate{
		Name:        "technical_research_team",
		Description: "A research and review team.",
		Instruction: new(`
					You coordinate a technical research team.
					
					CRITICAL RULES:
					1. Call the researcher tool to analyze the user's question.
					2. Do NOT print or return the researcher's raw output to the user.
					3. Call the reviewer tool to examine the researcher's output and produce a polished final answer.
					4. Return ONLY the reviewer's final answer. Nothing else.
					
					Never include intermediate output, research notes, or process explanations in your response.
					Only the reviewer's final polished answer should appear in your response.
					`),
		Mode:              llmagent.ModeChat,
		ModelName:         "gemma4:31b-cloud",
		ModelCredentialID: model.ID,
		CredentialSource:  enums.CredentialSourceAuto,
	}
	agent, err := tf.agentService.CreateRootAgent(context.Background(), agentCreate)
	require.NoError(tf.test, err)
	require.NotEmpty(tf.test, agent)
	require.Equal(tf.test, agentCreate.Name, agent.Name)
	require.Equal(tf.test, agentCreate.Description, agent.Description.String)
	require.Equal(tf.test, *agentCreate.Instruction, agent.Instruction.String)
	require.False(tf.test, agent.GlobalInstruction.Valid)
	require.Equal(tf.test, llmagent.ModeChat, agent.Mode)
	require.Equal(tf.test, agentCreate.ModelName, agent.ModelName.String)
	require.Equal(tf.test, model.ID, agent.ModelCredentialID)
	require.Equal(tf.test, enums.CredentialSourceAuto, agent.CredentialSource)
	return agent
}

func TestRun_OK(t *testing.T) {
	fixture := newTestFixture(t)
	user := fixture.createUser()
	model := fixture.createAppModel()
	rootAgent := fixture.createTechnicalResearchTeamRootAgent(model)
	fixture.createResearcherSubAgent(model, rootAgent.ID)
	fixture.createReviewerSubAgent(model, rootAgent.ID)

	sessionID, err := uuid.NewV7()
	require.NoError(t, err)
	chatSession, err := fixture.sessionsQuery.CreateSession(context.Background(), sessions.CreateSessionParams{
		ID:          sessionID,
		UserID:      user.ID,
		AppName:     "nexflow",
		RootAgentID: rootAgent.ID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, chatSession)

	agentRunner := New(fixture.agentService, fixture.modelService, fixture.sessionService, user.ID, "nexflow")
	req := Request{
		RootAgentID: rootAgent.ID,
		SessionID:   chatSession.ID,
		UserMessage: "Design a document editor like google doc.",
	}
	err = agentRunner.Run(context.Background(), req, nil)
	require.NoError(t, err)
}
