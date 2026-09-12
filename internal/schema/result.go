package schema

import "github.com/google/uuid"

type Result struct {
	Interrupted *bool     `json:"interrupted,omitempty,omitzero"`
	RunID       uuid.UUID `json:"runId"`
	SessionID   uuid.UUID `json:"sessionId"`
}
