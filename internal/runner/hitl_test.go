package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/tool/toolconfirmation"
	"google.golang.org/genai"
)

// buildInterruptEvent constructs a non-partial ADK event that carries a HITL
// confirmation request, mirroring how ADK surfaces one mid-run.
func buildInterruptEvent(callID, toolName string, hint string) *session.Event {
	original := &genai.FunctionCall{
		ID:   "orig-" + callID,
		Name: toolName,
		Args: map[string]any{"query": "probe"},
	}
	confirmationCall := &genai.FunctionCall{
		ID:   callID,
		Name: toolconfirmation.FunctionCallName,
		Args: map[string]any{
			"toolConfirmation":     map[string]any{"hint": hint},
			"originalFunctionCall": original,
		},
	}
	return &session.Event{
		Author:             "researcher",
		LongRunningToolIDs: []string{callID},
		LLMResponse: model.LLMResponse{
			Content: &genai.Content{
				Role:  genai.RoleModel,
				Parts: []*genai.Part{{FunctionCall: confirmationCall}},
			},
		},
	}
}

func TestDetectInterrupt_Positive(t *testing.T) {
	ev := buildInterruptEvent("call-1", "shorten_url", "Approve shortening this URL?")

	info, ok := detectInterrupt(ev)
	require.True(t, ok, "expected interrupt detection")
	require.Equal(t, "call-1", info.CallID)
	require.Equal(t, "shorten_url", info.ToolName)
	require.Equal(t, "Approve shortening this URL?", info.Hint)
}

func TestDetectInterrupt_IgnoresPartial(t *testing.T) {
	ev := buildInterruptEvent("call-1", "shorten_url", "hint")
	ev.LLMResponse.Partial = true

	_, ok := detectInterrupt(ev)
	require.False(t, ok, "partial streaming chunks must not trigger an interrupt")
}

func TestDetectInterrupt_IgnoresPlainEvents(t *testing.T) {
	ev := &session.Event{
		Author: "researcher",
		LLMResponse: model.LLMResponse{
			Content: &genai.Content{Parts: []*genai.Part{{Text: "just a text reply"}}},
		},
	}

	_, ok := detectInterrupt(ev)
	require.False(t, ok)
}

func TestBuildConfirmationResponseContent(t *testing.T) {
	content := buildConfirmationResponseContent("call-9", true, "Follow up on invoice #1042")

	require.Equal(t, genai.RoleUser, content.Role)
	require.Len(t, content.Parts, 2)
	fr := content.Parts[0].FunctionResponse
	require.NotNil(t, fr)
	require.Equal(t, "call-9", fr.ID)
	require.Equal(t, toolconfirmation.FunctionCallName, fr.Name)
	require.Equal(t, map[string]any{"confirmed": true}, fr.Response)
	require.Equal(t, "Follow up on invoice #1042", content.Parts[1].Text)
}

func TestBuildConfirmationResponseContent_Reject(t *testing.T) {
	content := buildConfirmationResponseContent("call-9", false, "")

	require.Len(t, content.Parts, 1)
	fr := content.Parts[0].FunctionResponse
	require.NotNil(t, fr)
	require.Equal(t, map[string]any{"confirmed": false}, fr.Response)
}
