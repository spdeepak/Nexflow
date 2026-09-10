package runner

import (
	"google.golang.org/genai"

	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/tool/toolconfirmation"
)

// InterruptInfo describes a Human-in-the-Loop confirmation request that ADK
// emitted during a run. It mirrors the wrapped confirmation function call so
// the UI can present an approve/reject prompt and later resume the run.
type InterruptInfo struct {
	// RunID is the interrupted run's id (empty if the runner set it).
	RunID string `json:"runId"`
	// CallID is the id of the "adk_request_confirmation" FunctionCall. A resume
	// must reply with a FunctionResponse carrying this same id.
	CallID string `json:"callId"`
	// ToolName is the name of the original tool the agent wanted to invoke.
	ToolName string `json:"toolName"`
	// Hint is the human-readable confirmation message, if any.
	Hint string `json:"hint"`
	// Args holds the original tool call arguments.
	Args map[string]any `json:"args"`
	// Message is optional extra context the operator appends when resuming an
	// interrupted run. It is not populated by detectInterrupt; callers set it
	// on resume. It is forwarded as a user text part alongside the
	// confirmation response so the model (and the resumed tool's context)
	// can see it.
	Message string `json:"message"`
	// Confirmed carries the operator's decision when resuming an interrupted
	// run. It is not populated by detectInterrupt; callers set it on resume.
	Confirmed bool `json:"-"`
}

// detectInterrupt reports whether a non-partial streamed event is an ADK HITL confirmation request.
// When true it also returns the populated InterruptInfo.
// Interrupts surface as an "adk_request_confirmation" FunctionCall whose id is listed in the event's LongRunningToolIDs;
// streaming partial chunks carry the same ids and must be deduped separately by the caller.
func detectInterrupt(event *session.Event) (InterruptInfo, bool) {
	if event == nil || event.LLMResponse.Partial || len(event.LongRunningToolIDs) == 0 {
		return InterruptInfo{}, false
	}
	longRunning := make(map[string]struct{}, len(event.LongRunningToolIDs))
	for _, id := range event.LongRunningToolIDs {
		longRunning[id] = struct{}{}
	}
	if event.Content == nil {
		return InterruptInfo{}, false
	}
	for _, part := range event.Content.Parts {
		if part == nil || part.FunctionCall == nil {
			continue
		}
		functionCall := part.FunctionCall
		if functionCall.Name != toolconfirmation.FunctionCallName {
			continue
		}
		if _, ok := longRunning[functionCall.ID]; !ok {
			continue
		}
		info := InterruptInfo{CallID: functionCall.ID}
		if original, err := toolconfirmation.OriginalCallFrom(functionCall); err == nil {
			info.ToolName = original.Name
			info.Args = original.Args
		}
		if args, ok := functionCall.Args["toolConfirmation"].(map[string]any); ok {
			if hint, hintPresent := args["hint"].(string); hintPresent {
				info.Hint = hint
			}
		}
		return info, true
	}
	return InterruptInfo{}, false
}

// buildConfirmationResponseContent builds a user-authored function response
// that resumes an interrupted confirmation. ADK's confirmation processor scans
// the session for such an event keyed by callID and either executes (confirmed)
// or blocks the suspended tool. When message is non-empty it is appended as a
// separate user text part so the follow-up turn can take the operator's note
// into account.
func buildConfirmationResponseContent(callID string, confirmed bool, message string) *genai.Content {
	parts := []*genai.Part{{
		FunctionResponse: &genai.FunctionResponse{
			ID:       callID,
			Name:     toolconfirmation.FunctionCallName,
			Response: map[string]any{"confirmed": confirmed},
		},
	}}
	if message != "" {
		parts = append(parts, &genai.Part{Text: message})
	}
	return &genai.Content{
		Role:  genai.RoleUser,
		Parts: parts,
	}
}
