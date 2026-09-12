package schema

import (
	"time"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/enums"
)

type SkillMetadata map[string]any

type Skill struct {
	AgentCount  int                    `json:"agentCount"`
	Content     *string                `json:"content,omitempty,omitzero"`
	ContentType enums.SkillContentType `json:"contentType"`
	CreatedAt   time.Time              `json:"createdAt"`
	ID          uuid.UUID              `json:"id"`
	IsActive    bool                   `json:"isActive"`
	Metadata    SkillMetadata          `json:"metadata"`
	Scope       enums.SkillScope       `json:"scope"`
	StorageUri  *string                `json:"storageUri,omitempty,omitzero"`
	Title       string                 `json:"title"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	UserID      uuid.UUID              `json:"userId"`
}

type SkillCreate struct {
	validate[SkillCreate]
	Content     *string                `json:"content,omitempty,omitzero"`
	ContentType enums.SkillContentType `json:"contentType"`
	Metadata    *SkillMetadata         `json:"metadata,omitempty,omitzero"`
	Scope       enums.SkillScope       `json:"scope"`
	StorageUri  *string                `json:"storageUri,omitempty,omitzero"`
	Title       string                 `json:"title"`
}

type SkillUpdate struct {
	validate[SkillUpdate]
	Content     *string                 `json:"content,omitempty,omitzero"`
	ContentType *enums.SkillContentType `json:"contentType,omitempty,omitzero"`
	IsActive    *bool                   `json:"isActive,omitempty,omitzero"`
	Metadata    *SkillMetadata          `json:"metadata,omitempty,omitzero"`
	StorageUri  *string                 `json:"storageUri,omitempty,omitzero"`
	Title       *string                 `json:"title,omitempty,omitzero"`
}
