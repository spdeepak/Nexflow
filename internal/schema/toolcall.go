package schema

import (
	"time"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/enums"
)

type ToolCall struct {
	AgentName  *string               `json:"agentName,omitempty,omitzero"`
	Arguments  *map[string]any       `json:"arguments,omitempty,omitzero"`
	CreatedAt  *time.Time            `json:"createdAt,omitempty,omitzero"`
	DurationMs *int                  `json:"durationMs,omitempty,omitzero"`
	EventID    *uuid.UUID            `json:"eventId,omitempty,omitzero"`
	ID         *uuid.UUID            `json:"id,omitempty,omitzero"`
	ResultJson *map[string]any       `json:"resultJson,omitempty,omitzero"`
	RunID      *uuid.UUID            `json:"runId,omitempty,omitzero"`
	Status     *enums.ToolCallStatus `json:"status,omitempty,omitzero"`
	ToolName   *string               `json:"toolName,omitempty,omitzero"`
}
