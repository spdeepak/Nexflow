package sessions

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/adk/v2/session"

	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/events"
	"github.com/spdeepak/nexflow/internal/invocation"
	"github.com/spdeepak/nexflow/internal/sessionstate"
)

// SessionStore is the ADK session.Service backed by the sessions, events and
// session_state tables. It persists events but does not itself emit UI events;
// the ChatService forwards events to the Wails runtime via the runner callback.
type SessionStore struct {
	sessions Querier
	events   events.Querier
	state    sessionstate.Querier
	appName  string
}

// NewSessionStore builds a DB-backed ADK session.Service.
func NewSessionStore(sessions Querier, events events.Querier, state sessionstate.Querier, appName string) session.Service {
	return &SessionStore{
		sessions: sessions,
		events:   events,
		state:    state,
		appName:  appName,
	}
}

func (s *SessionStore) Create(ctx context.Context, req *session.CreateRequest) (*session.CreateResponse, error) {
	if req.AppName == "" || req.UserID == "" {
		return nil, fmt.Errorf("app_name and user_id are required, got app_name: %q, user_id: %q", req.AppName, req.UserID)
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id %q: %w", req.UserID, err)
	}

	var id uuid.UUID
	if req.SessionID != "" {
		id, err = uuid.Parse(req.SessionID)
		if err != nil {
			return nil, fmt.Errorf("invalid session id %q: %w", req.SessionID, err)
		}
	} else {
		id, _ = uuid.NewV7()
	}

	// The root agent for a session created on the fly is unknown; the ChatService
	// creates explicit sessions with a root agent before running. Here we use the
	// zero uuid as a placeholder that is never exposed to the UI.
	created, err := s.sessions.CreateSession(ctx, CreateSessionParams{
		ID:          id,
		UserID:      userID,
		AppName:     req.AppName,
		RootAgentID: uuid.Nil,
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create session", "user", req.UserID, "error", err)
		return nil, err
	}

	return &session.CreateResponse{Session: &dbSession{
		sessionId:  created.ID,
		userID:     created.UserID,
		appName:    created.AppName,
		lastUpdate: created.LastUpdate,
		store:      s,
	}}, nil
}

func (s *SessionStore) Get(ctx context.Context, req *session.GetRequest) (*session.GetResponse, error) {
	id, err := uuid.Parse(req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session id %q: %w", req.SessionID, err)
	}
	row, err := s.sessions.GetSession(ctx, id)
	if err != nil {
		return nil, err
	}
	return &session.GetResponse{Session: &dbSession{
		sessionId:  row.ID,
		userID:     row.UserID,
		appName:    row.AppName,
		lastUpdate: row.LastUpdate,
		store:      s,
	}}, nil
}

func (s *SessionStore) List(ctx context.Context, req *session.ListRequest) (*session.ListResponse, error) {
	rows, err := s.sessions.ListSessions(ctx, ListSessionsParams{
		AppName: req.AppName,
		UserID:  req.UserID,
	})
	if err != nil {
		return nil, err
	}
	result := make([]session.Session, 0, len(rows))
	for _, row := range rows {
		result = append(result, &dbSession{
			sessionId:  row.ID,
			userID:     row.UserID,
			appName:    row.AppName,
			lastUpdate: row.LastUpdate,
			store:      s,
		})
	}
	return &session.ListResponse{Sessions: result}, nil
}

func (s *SessionStore) Delete(ctx context.Context, req *session.DeleteRequest) error {
	id, err := uuid.Parse(req.SessionID)
	if err != nil {
		return fmt.Errorf("invalid session id %q: %w", req.SessionID, err)
	}
	return s.sessions.DeleteSession(ctx, id)
}

func (s *SessionStore) AppendEvent(ctx context.Context, cur session.Session, ev *session.Event) error {
	if cur == nil || ev == nil {
		return fmt.Errorf("session and event are required")
	}
	// Partial (streaming chunk) events are not persisted.
	if ev.LLMResponse.Partial {
		return nil
	}
	sessID, err := uuid.Parse(cur.ID())
	if err != nil {
		return fmt.Errorf("invalid session id %q: %w", cur.ID(), err)
	}

	seq, err := s.events.NextEventSeq(ctx, sessID)
	if err != nil {
		return err
	}

	invocationID := sessionInvocation(ctx)
	contentJSON, err := marshalContent(ev)
	if err != nil {
		return err
	}
	actionsJSON, err := json.Marshal(ev.Actions)
	if err != nil {
		return err
	}
	role := eventRole(ev)

	eventID, _ := uuid.NewV7()
	tokenUsage, err := json.Marshal(ev.LLMResponse.UsageMetadata)
	if err != nil {
		slog.Error("Failed to marshal usage metadata", "event", ev, "error", err)
		tokenUsage = json.RawMessage("{}")
	}
	_, err = s.events.CreateEvent(ctx, events.CreateEventParams{
		ID:             eventID,
		SessionID:      sessID,
		InvocationID:   invocationID,
		Seq:            seq,
		Branch:         ev.Branch,
		IsolationScope: ev.IsolationScope,
		Author:         ev.Author,
		Role:           role,
		ContentJson:    contentJSON,
		ActionsJson:    actionsJSON,
		IsPartial:      ev.LLMResponse.Partial,
		IsFinal:        ev.IsFinalResponse(),
		TokenUsage:     tokenUsage,
		OutputJson:     marshalOutput(ev),
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to persist event", "session", sessID, "author", ev.Author, "error", err)
		return err
	}

	// Propagate session state deltas (temp:* keys are transient and skipped).
	if len(ev.Actions.StateDelta) > 0 {
		for key, value := range ev.Actions.StateDelta {
			if strings.HasPrefix(key, session.KeyPrefixTemp) {
				continue
			}
			raw, err := json.Marshal(value)
			if err != nil {
				return err
			}
			if err := s.state.SetSessionState(ctx, sessionstate.SetSessionStateParams{
				SessionID: sessID,
				Key:       key,
				Value:     raw,
			}); err != nil {
				slog.ErrorContext(ctx, "Failed to persist session state", "key", key, "error", err)
			}
		}
	}

	if err := s.sessions.UpdateSessionLastUpdate(ctx, sessID); err != nil {
		slog.WarnContext(ctx, "Failed to update session last_update", "session", sessID, "error", err)
	}
	return nil
}

// sessionInvocation returns the invocation uuid assigned to the active run by
// the ChatService. When no run has been created (e.g. sessions managed purely
// through the ADK service), events are grouped under a nil invocation.
func sessionInvocation(ctx context.Context) uuid.UUID {
	if id, ok := invocation.LookupInvocation(ctx); ok {
		return id
	}
	return uuid.Nil
}

// marshalContent serializes the event's content (the prompt/message parts).
func marshalContent(ev *session.Event) (json.RawMessage, error) {
	if ev.LLMResponse.Content == nil {
		return json.RawMessage{}, nil
	}
	return json.Marshal(ev.LLMResponse.Content)
}

func marshalOutput(ev *session.Event) json.RawMessage {
	switch v := ev.Output.(type) {
	case nil:
		return json.RawMessage("{}")
	case string:
		return json.RawMessage("{}") // plain text lives in content_json already
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return json.RawMessage("{}")
		}
		return b
	}
}

// eventRole derives the events.role enum value from an ADK event.
func eventRole(ev *session.Event) enums.EventRole {
	if ev.Author == "user" {
		return enums.EventRoleUser
	}
	if ev.LLMResponse.Content != nil {
		for _, part := range ev.LLMResponse.Content.Parts {
			if part == nil {
				continue
			}
			if part.FunctionCall != nil || part.FunctionResponse != nil {
				return enums.EventRoleFunction
			}
		}
	}
	if ev.Actions.TransferToAgent != "" {
		return enums.EventRoleFunction
	}
	return enums.EventRoleModel
}
