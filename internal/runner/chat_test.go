package runner

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/schema"
)

// TestServiceRun_StreamsSubAgentEvents exercises the full app path
// (Chat.Run -> runner.Run) and verifies the stage-event stream includes
// every producing agent — previously the stream stopped at the first
// sub-agent's final response, so reviewer/root-final events never reached the UI.
func TestServiceRun_StreamsSubAgentEvents(t *testing.T) {
	fixture := newTestFixture(t)
	user := fixture.createUser()

	ollama_key, _ := os.LookupEnv("OLLAMA_API_KEY")

	model, err := fixture.modelService.CreateUserModelCredential(context.Background(), user.ID, schema.ModelCredentialCreate{
		ApiKey:      ollama_key,
		BaseUrl:     "https://localhost:11434/v1",
		ExtraConfig: schema.ModelCredentialCreateExtraConfig{},
		Provider:    "ollama",
		ModelName:   "gemma4:31b-cloud",
		Scope:       enums.CredentialScopeUser,
		Title:       "Ollama gemma cloud",
	})
	require.NoError(t, err)

	rootAgent := fixture.createTechnicalResearchTeamRootAgent(model)
	fixture.createResearcherSubAgent(model, rootAgent.ID)
	fixture.createReviewerSubAgent(model, rootAgent.ID)

	chatService := NewChat(fixture.sessionsQuery, fixture.eventsQuery, fixture.runsQuery, fixture.sessionStateQuery, fixture.agentService, fixture.modelService, user.ID, "nexflow")

	session, err := chatService.CreateSession(context.Background(), user.ID, rootAgent.ID)
	require.NoError(t, err)

	emitted := map[string]bool{}
	res, err := chatService.Run(context.Background(), *session.ID, rootAgent.ID, "Design an editor like miro", Options{
		Emitter: func(name string, data any) {
			if re, ok := data.(RunEvent); ok {
				emitted[re.Agent] = true
			}
		},
	})
	require.NoError(t, err)

	require.True(t, emitted["researcher"], "researcher should have run")
	require.True(t, emitted["reviewer"], "reviewer should have run")
	require.True(t, emitted["technical_research_team"], "root should have run")

	run, err := fixture.runsQuery.GetRun(context.Background(), res.RunID)
	require.NoError(t, err)
	require.Equal(t, enums.RunStatusCompleted, run.Status)
}
