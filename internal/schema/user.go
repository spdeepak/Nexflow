package schema

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	CreatedAt  time.Time `json:"createdAt"`
	ExternalID string    `json:"externalId"`
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
}
