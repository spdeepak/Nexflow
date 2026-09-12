package runner

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/adk/v2/session"

	"github.com/spdeepak/nexflow/internal/agents"
	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/events"
	"github.com/spdeepak/nexflow/internal/invocation"
	"github.com/spdeepak/nexflow/internal/modelcredentials"
	"github.com/spdeepak/nexflow/internal/runs"
	"github.com/spdeepak/nexflow/internal/schema"
	"github.com/spdeepak/nexflow/internal/sessions"
	"github.com/spdeepak/nexflow/internal/sessionstate"
)

type (
	// Emitter delivers an event to the UI (Wails). name is the event name; data is JSON-serializable payload. May be nil-safe in tests.
	Emitter func(name string, data any)
	// Options configures a run.
	Options struct {
		Emitter Emitter
		// Resume, when set, resumes a prior interrupted run with a confirmation decision instead of starting a fresh user message.
		Resume *InterruptInfo
	}
	// Chat orchestrates chat runs and persists everything.
	Chat struct {
		sessionQuery            sessions.Querier
		eventsQuery             events.Querier
		runsQuery               runs.Querier
		sessionStateQuery       sessionstate.Querier
		agentService            agents.Service
		modelCredentialsService modelcredentials.Service
		userID                  uuid.UUID
		appName                 string
	}
	// RunEvent is the payload emitted to the UI for every streamed event
	RunEvent struct {
		Agent string         `json:"agent"`
		Event *session.Event `json:"event"`
		Final bool           `json:"final"`
	}
)

func NewChat(sessionQuery sessions.Querier, eventsQuery events.Querier, runsQuery runs.Querier, sessionStateQuery sessionstate.Querier, agentService agents.Service, modelCredentialsService modelcredentials.Service, userID uuid.UUID, appName string) *Chat {
	return &Chat{
		sessionQuery:            sessionQuery,
		eventsQuery:             eventsQuery,
		runsQuery:               runsQuery,
		sessionStateQuery:       sessionStateQuery,
		agentService:            agentService,
		modelCredentialsService: modelCredentialsService,
		userID:                  userID,
		appName:                 appName,
	}
}

func (c *Chat) Run(ctx context.Context, sessionID, rootAgentID uuid.UUID, userMessage string, opts Options) (schema.Result, error) {
	emit := opts.Emitter
	if emit == nil {
		emit = func(string, any) {}
	}

	runID, _ := uuid.NewV7()
	if _, err := c.runsQuery.CreateRun(ctx, runs.CreateRunParams{
		ID:           runID,
		SessionID:    sessionID,
		InvocationID: runID,
		RootAgentID:  rootAgentID,
	}); err != nil {
		return schema.Result{}, fmt.Errorf("failed to create run: %w", err)
	}

	sessionStore := sessions.NewSessionStore(c.sessionQuery, c.eventsQuery, c.sessionStateQuery, c.appName)

	chatRunner := New(c.agentService, c.modelCredentialsService, sessionStore, c.userID, c.appName)

	runCtx := invocation.ContextWithInvocation(ctx, runID)

	request := Request{
		RootAgentID:  rootAgentID,
		SessionID:    sessionID,
		UserMessage:  userMessage,
		Confirmation: opts.Resume,
	}
	onEventFunc := func(ev StageEvent) error {
		if ev.Interrupt != nil {
			interrupt := *ev.Interrupt
			interrupt.RunID = runID.String()
			emit("chat:interrupt", interrupt)
		}
		runEvent := RunEvent{
			Agent: ev.AgentName,
			Event: ev.Event,
			Final: ev.Final,
		}
		emit("chat:event", runEvent)
		return nil
	}
	chatRunError := chatRunner.Run(runCtx, request, onEventFunc)

	status := enums.RunStatusCompleted
	interrupted := false
	switch {
	case errors.Is(chatRunError, errInterrupted):
		status = enums.RunStatusInterrupted
		interrupted = true
		chatRunError = nil
	case chatRunError != nil:
		status = enums.RunStatusFailed
		slog.ErrorContext(ctx, "Chat run failed", "session", sessionID, "run", runID, "error", chatRunError)
	}

	_, statusErr := c.runsQuery.UpdateRunStatus(ctx, runs.UpdateRunStatusParams{
		ID:     runID,
		Status: status,
		Error:  nullString(chatRunError),
	})
	if statusErr != nil {
		slog.ErrorContext(ctx, "Failed to update run status", "run", runID, "error", statusErr)
	}

	if chatRunError != nil {
		return schema.Result{}, chatRunError
	}
	return schema.Result{SessionID: sessionID, RunID: runID, Interrupted: &interrupted}, nil
}

// CreateSession creates a chat thread for the given user and agent tree.
func (c *Chat) CreateSession(ctx context.Context, userID, rootAgentID uuid.UUID) (schema.Session, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return schema.Session{}, err
	}
	sess, err := c.sessionQuery.CreateSession(ctx, sessions.CreateSessionParams{
		ID:          id,
		UserID:      userID,
		AppName:     c.appName,
		RootAgentID: rootAgentID,
	})
	if err != nil {
		return schema.Session{}, err
	}
	return schema.Session{
		AppName:     sess.AppName,
		CreatedAt:   sess.CreatedAt,
		ID:          sess.ID,
		RootAgentID: sess.RootAgentID,
		UserID:      sess.UserID,
	}, nil
}

// GetSession returns a chat thread by id.
func (c *Chat) GetSession(ctx context.Context, id uuid.UUID) (sessions.Session, error) {
	return c.sessionQuery.GetSession(ctx, id)
}

// ListSessions returns the chat threads for a user, newest first.
func (c *Chat) ListSessions(ctx context.Context, userID uuid.UUID) ([]schema.Session, error) {
	if sessionList, err := c.sessionQuery.ListSessions(ctx, sessions.ListSessionsParams{
		UserID:  userID.String(),
		AppName: c.appName,
	}); err != nil {
		return nil, err
	} else {
		schemaSessions := make([]schema.Session, 0, len(sessionList))
		for _, sess := range sessionList {
			schemaSessions = append(schemaSessions, schema.Session{
				AppName:     sess.AppName,
				CreatedAt:   sess.CreatedAt,
				LastUpdate:  sess.LastUpdate,
				ID:          sess.ID,
				RootAgentID: sess.RootAgentID,
				UserID:      sess.UserID,
			})
		}
		return schemaSessions, nil
	}
}

// DeleteSession removes a chat thread and all of its events.
func (c *Chat) DeleteSession(ctx context.Context, id uuid.UUID) error {
	if err := c.sessionQuery.DeleteSession(ctx, id); err != nil {
		return err
	}
	return c.eventsQuery.DeleteSessionEvents(ctx, id)
}

// ListEvents returns a session's events in sequence order.
func (c *Chat) ListEvents(ctx context.Context, sessionID uuid.UUID) ([]schema.Event, error) {
	if eventsList, err := c.eventsQuery.ListSessionEvents(ctx, events.ListSessionEventsParams{
		Limit:        -1,
		Offset:       0,
		SessionID:    sessionID,
		InvocationID: nil,
		Author:       nil,
		Role:         nil,
	}); err != nil {
		return nil, err
	} else {
		schemaEvents := make([]schema.Event, 0, len(eventsList))
		for _, event := range eventsList {
			schemaEvents = append(schemaEvents, schema.Event{
				ID:             &event.ID,
				SessionID:      &event.SessionID,
				InvocationID:   &event.InvocationID,
				Seq:            &event.Seq,
				Branch:         &event.Branch,
				IsolationScope: &event.IsolationScope,
				Author:         &event.Author,
				Role:           &event.Role,
				ContentJson:    toEventJSON(event.ContentJson),
				ActionsJson:    toEventJSON(event.ActionsJson),
				IsPartial:      &event.IsPartial,
				IsFinal:        &event.IsFinal,
				TokenUsage:     toEventJSON(event.TokenUsage),
				OutputJson:     toEventJSON(event.OutputJson),
				CreatedAt:      &event.CreatedAt,
			})
		}
		return schemaEvents, nil
	}
}

// GetRun returns a run by id.
func (c *Chat) GetRun(ctx context.Context, id uuid.UUID) (schema.Run, error) {
	if run, err := c.runsQuery.GetRun(ctx, id); err != nil {
		return schema.Run{}, err
	} else {
		var finishedAt *time.Time
		if run.FinishedAt.Valid {
			finishedAt = &run.FinishedAt.Time
		}
		var runError string
		if run.Error.Valid {
			runError = run.Error.String
		}
		return schema.Run{
			ID:           &run.ID,
			SessionID:    &run.SessionID,
			InvocationID: &run.InvocationID,
			RootAgentID:  &run.RootAgentID,
			Status:       &run.Status,
			StartedAt:    &run.StartedAt,
			FinishedAt:   finishedAt,
			Error:        &runError,
		}, nil
	}
}

// nullString converts an optional run error to a sql.NullString.
func nullString(err error) sql.NullString {
	if err == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: err.Error(), Valid: true}
}

func toEventJSON(msg json.RawMessage) *map[string]any {
	m := map[string]interface{}{}
	_ = json.Unmarshal(msg, &m)
	return &m
}
