package sessions

import (
	"context"
	"encoding/json"
	"iter"

	"github.com/google/uuid"
	"google.golang.org/adk/v2/session"

	"github.com/spdeepak/nexflow/internal/sessionstate"
)

// dbState adapts the session_state table to session.State.
type dbState struct {
	sessionID uuid.UUID
	store     *SessionStore
}

func (st *dbState) Get(key string) (any, error) {
	ctx := context.Background()
	row, err := st.store.state.GetSessionStateKey(ctx, sessionstate.GetSessionStateKeyParams{
		SessionID: st.sessionID,
		Key:       key,
	})
	if err != nil {
		return nil, session.ErrStateKeyNotExist
	}
	var value any
	if err := json.Unmarshal(row.Value, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func (st *dbState) Set(key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return st.store.state.SetSessionState(context.Background(), sessionstate.SetSessionStateParams{
		SessionID: st.sessionID,
		Key:       key,
		Value:     raw,
	})
}

func (st *dbState) All() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		rows, err := st.store.state.GetSessionState(context.Background(), st.sessionID)
		if err != nil {
			return
		}
		for _, row := range rows {
			var value any
			if err := json.Unmarshal(row.Value, &value); err != nil {
				continue
			}
			if !yield(row.Key, value) {
				return
			}
		}
	}
}
