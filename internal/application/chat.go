package application

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/spdeepak/nexflow/internal/runner"
	"github.com/spdeepak/nexflow/internal/schema"
)

// currentUser returns the user bound to this device.
func (a *App) currentUser(ctx context.Context) (uuid.UUID, error) {
	row, err := a.userService.GetUserByExternalID(ctx, a.deviceID.String())
	if err != nil {
		return uuid.Nil, err
	}
	return row.ID, nil
}

// emit forwards a Wails event to the frontend.
func (a *App) emit(name string, data any) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, name, data)
}

// CreateSession creates a new chat thread for an agent tree.
func (a *App) CreateSession(rootAgentID string) (schema.Session, error) {
	root, err := uuid.Parse(rootAgentID)
	if err != nil {
		return schema.Session{}, fmt.Errorf("invalid root agent ID: %w", err)
	}
	userID, err := a.currentUser(a.ctx)
	if err != nil {
		return schema.Session{}, fmt.Errorf("failed to resolve current user: %w", err)
	}
	sess, err := a.chatService.CreateSession(a.ctx, userID, root)
	if err != nil {
		slog.ErrorContext(a.ctx, "Failed to create session", "error", err)
		return schema.Session{}, err
	}
	return sess, nil
}

// ListSessions lists the current user's chat threads, newest first.
func (a *App) ListSessions() ([]schema.Session, error) {
	userID, err := a.currentUser(a.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve current user: %w", err)
	}
	sessionsList, err := a.chatService.ListSessions(a.ctx, userID)
	if err != nil {
		slog.ErrorContext(a.ctx, "Failed to list sessions", "error", err)
		return nil, err
	}
	return sessionsList, nil
}

// DeleteSession deletes a chat thread and all its events.
func (a *App) DeleteSession(id string) error {
	sessionID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}
	return a.chatService.DeleteSession(a.ctx, sessionID)
}

// SendMessageSession runs the user's message through the session's agent tree,
// streaming chat:event updates to the frontend as stages execute.
func (a *App) SendMessageSession(sessionID string, message string) (schema.Result, error) {
	parsedSessionID, err := uuid.Parse(sessionID)
	if err != nil {
		return schema.Result{}, fmt.Errorf("invalid session ID: %w", err)
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return schema.Result{}, fmt.Errorf("message cannot be empty")
	}

	sess, err := a.chatService.GetSession(a.ctx, parsedSessionID)
	if err != nil {
		return schema.Result{}, fmt.Errorf("session not found: %w", err)
	}

	if result, err := a.chatService.Run(a.ctx, parsedSessionID, sess.RootAgentID, message, runner.Options{Emitter: a.emit}); err != nil {
		return schema.Result{}, err
	} else {
		return result, nil
	}
}

// ConfirmSession resumes an interrupted chat run by submitting the
// operator's approve/reject decision for a pending Human-in-the-Loop
// confirmation. callID is the id of the "adk_request_confirmation" request
// returned in the "chat:interrupt" event. message is optional extra context
// the operator wants the resumed turn to take into account.
func (a *App) ConfirmSession(sessionID string, callID string, confirmed bool, message string) (schema.Result, error) {
	parsedSessionID, err := uuid.Parse(sessionID)
	if err != nil {
		return schema.Result{}, fmt.Errorf("invalid session ID: %w", err)
	}
	if callID == "" {
		return schema.Result{}, fmt.Errorf("confirmation call ID cannot be empty")
	}

	sess, err := a.chatService.GetSession(a.ctx, parsedSessionID)
	if err != nil {
		return schema.Result{}, fmt.Errorf("session not found: %w", err)
	}

	if result, err := a.chatService.Run(a.ctx, parsedSessionID, sess.RootAgentID, "", runner.Options{
		Emitter: a.emit,
		Resume: &runner.InterruptInfo{
			CallID:    callID,
			Confirmed: confirmed,
			Message:   message,
		},
	}); err != nil {
		return schema.Result{}, err
	} else {
		return result, nil
	}
}

// ListEvents returns a session's events in sequence order.
func (a *App) ListEvents(sessionID string) ([]schema.Event, error) {
	parsed, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}
	rows, err := a.chatService.ListEvents(a.ctx, parsed)
	if err != nil {
		slog.ErrorContext(a.ctx, "Failed to list events", "error", err)
		return nil, err
	}
	return rows, nil
}

// GetRun returns a run by id.
func (a *App) GetRun(runID string) (schema.Run, error) {
	parsed, err := uuid.Parse(runID)
	if err != nil {
		return schema.Run{}, fmt.Errorf("invalid run ID: %w", err)
	}
	run, err := a.chatService.GetRun(a.ctx, parsed)
	if err != nil {
		slog.ErrorContext(a.ctx, "Failed to get run", "error", err)
		return schema.Run{}, err
	}
	return run, nil
}
