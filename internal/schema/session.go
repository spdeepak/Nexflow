package schema

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	AppName     string    `json:"appName,omitempty,omitzero"`
	CreatedAt   time.Time `json:"createdAt,omitzero"`
	ID          uuid.UUID `json:"id,omitempty,omitzero"`
	LastUpdate  time.Time `json:"lastUpdate,omitzero"`
	RootAgentID uuid.UUID `json:"rootAgentId,omitempty,omitzero"`
	UserID      uuid.UUID `json:"userId,omitempty,omitzero"`
}

type SessionCreate struct {
	validate[SessionCreate]
	AppName     string    `json:"appName" validate:"required"`
	RootAgentID uuid.UUID `json:"rootAgentId" validate:"required"`
	UserID      uuid.UUID `json:"userId" validate:"required"`
}
