package schema

import (
	"time"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/enums"
)

type Event struct {
	ActionsJson    *map[string]any  `json:"actionsJson,omitempty,omitzero"`
	Author         *string          `json:"author,omitempty,omitzero"`
	Branch         *string          `json:"branch,omitempty,omitzero"`
	ContentJson    *map[string]any  `json:"contentJson,omitempty,omitzero"`
	CreatedAt      *time.Time       `json:"createdAt,omitempty,omitzero"`
	ID             *uuid.UUID       `json:"id,omitempty,omitzero"`
	InvocationID   *uuid.UUID       `json:"invocationId,omitempty,omitzero"`
	IsFinal        *bool            `json:"isFinal,omitempty,omitzero"`
	IsPartial      *bool            `json:"isPartial,omitempty,omitzero"`
	IsolationScope *string          `json:"isolationScope,omitempty,omitzero"`
	OutputJson     *map[string]any  `json:"outputJson,omitempty,omitzero"`
	Role           *enums.EventRole `json:"role,omitempty,omitzero"`
	Seq            *int64           `json:"seq,omitempty,omitzero"`
	SessionID      *uuid.UUID       `json:"sessionId,omitempty,omitzero"`
	TokenUsage     *map[string]any  `json:"tokenUsage,omitempty,omitzero"`
}
