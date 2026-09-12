package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	adkagent "google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	adkauth "google.golang.org/adk/v2/auth"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/model/openaimodel"
	adkrunner "google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/mcptoolset"
	"google.golang.org/adk/v2/tool/skilltoolset"
	"google.golang.org/genai"

	"github.com/spdeepak/nexflow/internal/agents"
	"github.com/spdeepak/nexflow/internal/agentskills"
	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/modelcredentials"
	"github.com/spdeepak/nexflow/internal/schema"
)

// errInterrupted signals that a stage was halted because a HITL confirmation was requested. The run is left in a resumable (interrupted) state.
var errInterrupted = fmt.Errorf("run interrupted: human confirmation required")

// maxDelegationRetryAttempts bounds how many times the root agent is re-run after answering directly without invoking its sub-agents.
const maxDelegationRetryAttempts = 2

type (
	// StageEvent is one ADK event emitted while a stage agent runs.
	// AgentName is the name of the stage agent that produced it (parent for the plan/final passes, or a sub-agent for its sequential pass).
	// Final marks the last event of this stage agent (i.e. the run of that stage has completed and the next stage may begin).
	// Interrupt, when non-nil, signals a Human-in-the-Loop confirmation request. The current stage should stop streaming so the run can be paused (status=interrupted) and later resumed with a user decision.
	// This is used to show which agent card is supposed to glow in the application chat section
	StageEvent struct {
		AgentName string
		Event     *session.Event
		Final     bool
		Interrupt *InterruptInfo
	}

	// OnEventFunc receives stage events as they stream. Returning an error aborts the current stage run.
	OnEventFunc func(StageEvent) error

	// Request describes a single orchestrated run of the agent tree.
	Request struct {
		RootAgentID uuid.UUID
		SessionID   uuid.UUID
		UserMessage string
		// Confirmation is an optional HITL resume. When set, the run starts by
		// replaying a user-authored confirmation response (from a prior
		// interrupted run) so the previously suspended tool resumes.
		Confirmation *InterruptInfo
	}

	// Option configures a Run call.
	Option     func(*runOptions)
	runOptions struct{}

	// Runner orchestrates execution of an agent tree through ADK.
	//
	// It builds a root agent with sub-agents attached via ADK's native SubAgents
	// mechanism, then runs the root agent once — ADK handles sub-agent delegation
	// autonomously based on the root agent's instructions and sub-agent descriptions.
	runner struct {
		agentService            agents.Service
		modelCredentialsService modelcredentials.Service
		sessionService          session.Service
		skillsService           agentskills.Service
		userID                  uuid.UUID
		appName                 string
	}
	Runner interface {
		Run(ctx context.Context, req Request, onEvent OnEventFunc, opts ...Option) error
	}
)

func New(agentService agents.Service, modelCredentialService modelcredentials.Service, sessionService session.Service, userId uuid.UUID, appName string) Runner {
	return &runner{
		agentService:            agentService,
		modelCredentialsService: modelCredentialService,
		sessionService:          sessionService,
		userID:                  userId,
		appName:                 appName,
	}
}

// Run builds a root agent with its sub-agents wired via ADK's SubAgents mechanism, then executes the root agent.
// ADK autonomously delegates to sub-agents based on the root agent's instructions and the sub-agent descriptions.
//
// onEvent is invoked for every streamed event. Events carry the producing agent's name so the UI can live-highlight the active stage.
func (r *runner) Run(ctx context.Context, req Request, onEvent OnEventFunc, opts ...Option) error {
	options := &runOptions{}
	for _, o := range opts {
		o(options)
	}

	rootAgent, err := r.agentService.GetAgentDetail(ctx, req.RootAgentID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to resolve root agent for run", "rootAgentID", req.RootAgentID, "error", err)
		return err
	}

	subAgents, err := r.agentService.GetSubAgents(ctx, rootAgent.ID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get sub-agents for root agent", "rootAgentId", rootAgent.ID, "error", err)
		return err
	}

	adkSubAgents := make([]adkagent.Agent, len(subAgents))
	for index, sub := range subAgents {
		adkSubAgents[index], err = r.buildSubAgent(ctx, sub)
		if err != nil {
			return err
		}
	}

	rootADKAgent, err := r.buildRootAgentWithSubAgents(ctx, rootAgent, adkSubAgents)
	if err != nil {
		return err
	}

	sessionID := req.SessionID.String()
	userID := r.userID.String()

	subAgentNames := make(map[string]bool, len(adkSubAgents))
	for _, sub := range adkSubAgents {
		subAgentNames[sub.Name()] = true
	}

	// wrapEvent tracks whether any sub-agent actually produced an event during this run.
	// The root LLM intermittently answers directly and skips its tools; when that happens we re-run the root with a delegation directive.
	// It also captures HITL confirmation requests so the run can be paused.
	var invokedSubAgent bool
	var interrupt *InterruptInfo
	wrapEvent := func(se StageEvent) error {
		if se.Interrupt != nil {
			interrupt = se.Interrupt
			if onEvent != nil {
				return onEvent(se)
			}
			return errInterrupted
		}
		if subAgentNames[se.AgentName] {
			invokedSubAgent = true
		}
		if onEvent != nil {
			return onEvent(se)
		}
		return nil
	}

	// buildInput yields the content to feed the first stage: either the user's
	// prompt or, when resuming, a user-authored confirmation response.
	var rootInput *genai.Content
	if req.Confirmation != nil {
		rootInput = buildConfirmationResponseContent(req.Confirmation.CallID, req.Confirmation.Confirmed, req.Confirmation.Message)
	} else {
		rootInput = &genai.Content{
			Role:  genai.RoleUser,
			Parts: []*genai.Part{{Text: req.UserMessage}},
		}
	}

	for attempt := 0; attempt <= maxDelegationRetryAttempts; attempt++ {
		stageAgent := rootADKAgent
		if attempt > 0 {
			enforced, err := r.buildRootAgentWithDelegationDirective(ctx, rootAgent, adkSubAgents, attempt)
			if err != nil {
				return err
			}
			stageAgent = enforced
			slog.Info("Re-running root with delegation directive after direct answer", "rootAgent", rootAgent.Name, "attempt", attempt)
		}
		if _, err = r.runStage(ctx, stageAgent, sessionID, userID, rootInput, wrapEvent); err != nil {
			return err
		}
		// A confirmation was requested: the run is intentionally paused.
		if interrupt != nil {
			return errInterrupted
		}
		if invokedSubAgent {
			return nil
		}
		slog.WarnContext(ctx, "Root agent answered without delegating to sub-agents", "rootAgent", rootAgent.Name, "attempt", attempt)
	}

	return nil
}

// runStage runs one stage agent over the shared ADK session with the given input,
// streams its events to onEvent, and returns the stage's final text
// output so it can be fed to the next stage.
func (r *runner) runStage(ctx context.Context, adkAgent adkagent.Agent, sessionID, userID string, input *genai.Content, onEvent OnEventFunc) (string, error) {
	adkRunner, err := adkrunner.New(adkrunner.Config{
		AppName:        r.appName,
		Agent:          adkAgent,
		SessionService: r.sessionService,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create adk runner for %s: %w", adkAgent.Name(), err)
	}

	final := ""
	done := false
	for event, runErr := range adkRunner.Run(ctx, userID, sessionID, input, adkagent.RunConfig{}) {
		if runErr != nil {
			return "", runErr
		}
		if event == nil {
			continue
		}
		if done {
			// The root agent's final response has been delivered; keep
			// pulling the stream to exhaustion so the producing agent's
			// generator unwinds cleanly, but ignore trailing engine events.
			continue
		}
		agentName := adkAgent.Name()
		if event.Author != "" && event.Author != "user" {
			agentName = event.Author
		}
		finalEvent := event.IsFinalResponse()
		if onEvent != nil {
			var interrupt *InterruptInfo
			if info, ok := detectInterrupt(event); ok {
				interrupt = &info
			}
			se := StageEvent{AgentName: agentName, Event: event, Final: finalEvent, Interrupt: interrupt}
			if err = onEvent(se); err != nil {
				return "", err
			}
			// A HITL confirmation request halts this stage.
			if interrupt != nil {
				break
			}
		}
		if !event.LLMResponse.Partial {
			if text := extractText(event); text != "" {
				final = text
			}
		}
		// The orchestration only ends when the ROOT agent produces its final
		// response. A sub-agent's final response merely hands control back to
		// the root, so we keep streaming (reviewer + final answer included).
		if finalEvent && agentName == adkAgent.Name() {
			// Don't return yet: keep pulling the iterator to exhaustion so the producing agent's generator unwinds cleanly.
			done = true
		}
	}
	return final, nil
}

// buildSubAgent resolves the agent's model and constructs an ADK llmagent,
// deriving the agent's Mode, description and LLM generation config from the
// configuration stored on the agent.
func (r *runner) buildSubAgent(ctx context.Context, subAgent schema.Agent) (adkagent.Agent, error) {
	apiAgent, err := r.agentService.GetAgentDetail(ctx, subAgent.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get detail for sub-agent %s: %w", subAgent.Name, err)
	}

	agentModel, err := r.resolveModel(ctx, apiAgent)
	if err != nil {
		return nil, err
	}

	skills, err := r.resolveSkills(ctx, apiAgent)
	if err != nil {
		return nil, err
	}

	mcpToolsets := r.resolveMCPs(ctx, apiAgent)
	toolsets := make([]tool.Toolset, 0, 1+len(mcpToolsets))
	toolsets = append(toolsets, skills)
	toolsets = append(toolsets, mcpToolsets...)

	cfg := llmagent.Config{
		Name:                  apiAgent.Name,
		Description:           apiAgent.Description,
		Instruction:           apiAgent.Instruction,
		Model:                 agentModel,
		Mode:                  apiAgent.Mode,
		GlobalInstruction:     apiAgent.GlobalInstruction,
		GenerateContentConfig: &apiAgent.ModelConfig,
		Toolsets:              toolsets,
	}

	agent, err := llmagent.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build llm agent %s: %w", apiAgent.Name, err)
	}
	return agent, nil
}

// buildRootAgentWithSubAgents is like buildSubAgent but wires the given sub-agents into the root agent's config so ADK can delegate to them.
func (r *runner) buildRootAgentWithSubAgents(ctx context.Context, rootAgent schema.AgentDetail, adkSubAgents []adkagent.Agent) (adkagent.Agent, error) {
	agentModel, err := r.resolveModel(ctx, rootAgent)
	if err != nil {
		return nil, err
	}
	skills, err := r.resolveSkills(ctx, rootAgent)
	if err != nil {
		return nil, err
	}

	mcpToolsets := r.resolveMCPs(ctx, rootAgent)
	toolsets := make([]tool.Toolset, 0, 1+len(mcpToolsets))
	toolsets = append(toolsets, skills)
	toolsets = append(toolsets, mcpToolsets...)

	cfg := llmagent.Config{
		Name:                  rootAgent.Name,
		Description:           rootAgent.Description,
		Instruction:           rootAgent.Instruction,
		Model:                 agentModel,
		Mode:                  rootAgent.Mode,
		GlobalInstruction:     rootAgent.GlobalInstruction,
		GenerateContentConfig: &rootAgent.ModelConfig,
		Toolsets:              toolsets,
	}

	if len(adkSubAgents) > 0 {
		cfg.SubAgents = adkSubAgents
	}

	slog.Info("LLM agent config", "cfg", cfg, "instruction", cfg.Instruction)

	agent, err := llmagent.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build llm agent %s: %w", rootAgent.Name, err)
	}
	return agent, nil
}

// buildRootAgentWithDelegationDirective rebuilds the root agent with an escalated instruction that demands sub-agent use.
// It is used when the root LLM answered directly without invoking its sub-agents, which the model
// occasionally does despite its standing instructions.
func (r *runner) buildRootAgentWithDelegationDirective(ctx context.Context, rootAgent schema.AgentDetail, adkSubAgents []adkagent.Agent, attempt int) (adkagent.Agent, error) {
	names := make([]string, len(adkSubAgents))
	for i, sub := range adkSubAgents {
		names[i] = sub.Name()
	}
	directive := fmt.Sprintf(
		"\n\nIMPORTANT OVERRIDE (attempt %d): You MUST complete this request by calling your sub-agent tools. "+
			"Do not answer directly. Call the following tools in order: %s. "+
			"Only after you have called each tool and received its result should you produce your final answer.",
		attempt, strings.Join(names, ", "))

	clone := rootAgent
	clone.Instruction = rootAgent.Instruction + directive
	clone.GlobalInstruction = clone.GlobalInstruction + directive
	return r.buildRootAgentWithSubAgents(ctx, clone, adkSubAgents)
}

// resolveModel picks a model for the given agent spec based on its credential
// source, builds an OpenAI-compatible model client.
//
// Credential resolution:
//   - Explicit ModelCredentialID always wins.
//   - auto: prefer a user key, fall back to any available (user or app) key.
//   - user: only a user-scoped key.
//   - app: only an app-scoped key.
func (r *runner) resolveModel(ctx context.Context, apiAgent schema.AgentDetail) (model.LLM, error) {
	cred, err := r.getModelCredential(ctx, apiAgent)
	if err != nil {
		return nil, err
	}

	modelName := cred.ModelName
	if apiAgent.ModelName != "" {
		modelName = apiAgent.ModelName
	}
	var llm model.LLM
	if cred.ApiKeyCipher.Valid && cred.ApiKeyCipher.String != "" {
		llm, err = openaimodel.NewModel(ctx, modelName, &openaimodel.ClientConfig{
			BaseURL: normalizeBaseURL(cred.BaseUrl.String),
		})
	} else {
		// Providers like Ollama do not require an API key but expect a
		// non-empty Authorization header, so use the same placeholder that the
		// reference agent uses.
		llm, err = openaimodel.NewModel(ctx, modelName, &openaimodel.ClientConfig{
			APIKey:  cred.ApiKeyCipher.String,
			BaseURL: normalizeBaseURL(cred.BaseUrl.String),
		})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create model %s: %w", modelName, err)
	}
	// Transparently retry vacuous generations (empty/no text/tool content) that
	// local providers intermittently return instead of a real response.
	return withGenerateRetry(llm, defaultGenerateRetries), nil
}

func (r *runner) resolveSkills(ctx context.Context, apiAgent schema.AgentDetail) (*skilltoolset.SkillToolset, error) {
	inMemorySkills := make([]InMemorySkill, len(apiAgent.Skills))
	for index, skill := range apiAgent.Skills {
		inMemorySkills[index] = InMemorySkill{
			Name:         skill.Title,
			Description:  *skill.Content,
			Instructions: *skill.Content,
		}
	}
	return skilltoolset.New(ctx, skilltoolset.Config{
		Source: NewStringToolSet(inMemorySkills...),
	})
}

func (r *runner) resolveMCPs(ctx context.Context, apiAgent schema.AgentDetail) []tool.Toolset {
	agentMCPs := make([]tool.Toolset, len(apiAgent.Mcps))
	for index, agentMCP := range apiAgent.Mcps {
		mcpConfig := mcptoolset.Config{
			RequireConfirmation: agentMCP.RequireConfirmation,
		}
		switch agentMCP.Transport {
		case enums.McpTransportStdIO:
			cmd := exec.Command(*agentMCP.Command, agentMCP.Args...)
			mcpConfig.Transport = &mcp.CommandTransport{Command: cmd}
		default: // streamable_http
			mcpConfig.Endpoint = *agentMCP.Endpoint
		}

		if agentMCP.AuthType != nil && agentMCP.AuthConfig != nil {
			switch *agentMCP.AuthType {
			case enums.McpAuthTypeApiKey:
				if apiKey, ok := agentMCP.AuthConfig["apiKey"].(string); ok {
					mcpConfig.Auth = adkauth.StaticToken(apiKey)
				}
			case enums.McpAuthTypeOAUTH:
				if accessToken, ok := agentMCP.AuthConfig["accessToken"].(string); ok {
					mcpConfig.Auth = adkauth.StaticToken(accessToken)
				}
			}
		}

		mcpTool, err := mcptoolset.New(mcpConfig)
		if err != nil {
			slog.WarnContext(ctx, "Failed to create MCP toolset", "name", agentMCP.Name, "error", err)
			continue
		}
		if len(agentMCP.AllowedTools) > 0 {
			mcpTool = tool.FilterToolset(mcpTool, tool.AllowedToolsPredicate(agentMCP.AllowedTools))
		}
		agentMCPs[index] = mcpTool
	}
	return agentMCPs
}

func (r *runner) getModelCredential(ctx context.Context, apiAgent schema.AgentDetail) (modelcredentials.ModelCredential, error) {
	if apiAgent.ModelCredentialID != uuid.Nil {
		cred, err := r.modelCredentialsService.GetModelCredentialWithKey(ctx, apiAgent.ModelCredentialID)
		if err == nil {
			return cred, nil
		}
		slog.WarnContext(ctx, "Explicit credential lookup failed, falling back by source", "id", apiAgent.ModelCredentialID, "error", err)
	}

	active := true
	available, err := r.modelCredentialsService.GetAvailableModelCredentials(ctx, r.userID, &active)
	if err != nil {
		return modelcredentials.ModelCredential{}, fmt.Errorf("no credentials available to run agent %s: %w", apiAgent.Name, err)
	}

	var fallback schema.ModelCredential
	for _, c := range available {
		if !credentialMatchesSource(c.Scope, apiAgent.CredentialSource) {
			continue
		}
		if c.IsActive {
			return r.withKey(ctx, c)
		}
		if fallback.ID == uuid.Nil {
			fallback = c
		}
	}
	if fallback.ID != uuid.Nil {
		return r.withKey(ctx, fallback)
	}
	return modelcredentials.ModelCredential{}, fmt.Errorf("no active credential matching source %q found to run agent %s", apiAgent.CredentialSource, apiAgent.Name)
}

// withKey resolves the API key for a candidate credential (the listing query
// omits the key; only GetModelCredentialWithKey returns it).
func (r *runner) withKey(ctx context.Context, c schema.ModelCredential) (modelcredentials.ModelCredential, error) {
	if c.ID == uuid.Nil {
		return modelcredentials.ModelCredential{}, fmt.Errorf("cannot resolve key for empty credential")
	}
	cred, err := r.modelCredentialsService.GetModelCredentialWithKey(ctx, c.ID)
	if err != nil {
		return modelcredentials.ModelCredential{}, fmt.Errorf("failed to load credential %s key: %w", c.ID, err)
	}
	return cred, nil
}

// credentialMatchesSource filters available credentials by the agent's
// credential_source. auto accepts any scope (user preferred, app fallback);
// user requires a user scope; app requires an app scope.
func credentialMatchesSource(scope enums.CredentialScope, source enums.CredentialSource) bool {
	switch source {
	case enums.CredentialSourceUser:
		return scope == enums.CredentialScopeUser
	case enums.CredentialSourceApp:
		return scope == enums.CredentialScopeApp
	default: // auto
		return true
	}
}
