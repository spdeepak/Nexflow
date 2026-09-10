package sessions

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"

	"github.com/spdeepak/nexflow/internal/events"
)

// dbSession adapts a stored session to session.Session.
// This is required to satisfy the session.Session interface from google.golang.org/adk/v2/session.
// The ADK runtime itself calls Events() internally (e.g., when an agent reads prior conversation history from a session).
type dbSession struct {
	sessionId  uuid.UUID
	userID     uuid.UUID
	appName    string
	lastUpdate time.Time
	store      *SessionStore
}

func (s *dbSession) ID() string {
	return s.sessionId.String()
}

func (s *dbSession) AppName() string {
	return s.appName
}

func (s *dbSession) UserID() string {
	return s.userID.String()
}

func (s *dbSession) LastUpdateTime() time.Time {
	return s.lastUpdate
}

func (s *dbSession) State() session.State {
	return &dbState{sessionID: s.sessionId, store: s.store}
}

func (s *dbSession) Events() session.Events {
	ctx := context.Background()
	listEventParams := events.ListSessionEventsParams{
		Limit:        -1,
		Offset:       0,
		SessionID:    s.sessionId,
		InvocationID: nil,
		Author:       nil,
		Role:         nil,
	}
	list, err := s.store.events.ListSessionEvents(ctx, listEventParams)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to load session events", "session", s.sessionId, "error", err)
		return &eventList{}
	}
	result := make([]*session.Event, 0, len(list))
	for _, row := range list {
		ev := dbEventFromRow(row)
		result = append(result, ev)
	}
	return &eventList{items: result}
}

// dbEventFromRow reconstructs an ADK event from a stored events row.
func dbEventFromRow(row events.Event) *session.Event {
	ev := &session.Event{
		ID:             "e-" + row.ID.String(),
		Timestamp:      row.CreatedAt,
		InvocationID:   "e-" + row.InvocationID.String(),
		Branch:         row.Branch,
		Author:         row.Author,
		IsolationScope: row.IsolationScope,
	}
	if len(row.ContentJson) > 0 {
		var content genai.Content
		if err := json.Unmarshal(row.ContentJson, &content); err == nil {
			ev.LLMResponse.Content = &content
		}
	}
	if len(row.ActionsJson) > 0 {
		var actions session.EventActions
		if err := json.Unmarshal(row.ActionsJson, &actions); err == nil {
			ev.Actions = actions
		}
	}
	return ev
}
