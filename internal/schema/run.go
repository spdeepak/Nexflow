package schema

import (
	"time"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/enums"
)

type Run struct {
	Error        *string          `json:"error,omitempty,omitzero"`
	Events       []Event          `json:"events,omitempty,omitzero"`
	FinishedAt   *time.Time       `json:"finishedAt,omitempty,omitzero"`
	ID           *uuid.UUID       `json:"id,omitempty,omitzero"`
	InvocationID *uuid.UUID       `json:"invocationId,omitempty,omitzero"`
	RootAgentID  *uuid.UUID       `json:"rootAgentId,omitempty,omitzero"`
	SessionID    *uuid.UUID       `json:"sessionId,omitempty,omitzero"`
	StartedAt    *time.Time       `json:"startedAt,omitempty,omitzero"`
	Status       *enums.RunStatus `json:"status,omitempty,omitzero"`
}

type RunCreate struct {
	validate[RunCreate]
	Message string `json:"message" validate:"required"`
}
