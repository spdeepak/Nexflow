package schema

import (
	"time"

	"github.com/google/uuid"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/genai"

	"github.com/spdeepak/nexflow/internal/enums"
)

type ConfigJson map[string]any

type Agent struct {
	ConfigJson        ConfigJson                  `json:"configJson,omitempty,omitzero"`
	CreatedAt         time.Time                   `json:"createdAt"`
	CredentialSource  enums.CredentialSource      `json:"credentialSource"`
	Description       string                      `json:"description"`
	GlobalInstruction string                      `json:"globalInstruction,omitempty,omitzero"`
	ID                uuid.UUID                   `json:"id"`
	Instruction       string                      `json:"instruction"`
	IsActive          bool                        `json:"isActive"`
	Mode              llmagent.Mode               `json:"mode"`
	ModelConfig       genai.GenerateContentConfig `json:"modelConfig,omitzero"`
	ModelCredentialID uuid.UUID                   `json:"modelCredentialId"`
	ModelName         string                      `json:"modelName"`
	Name              string                      `json:"name"`
	ParentAgentID     uuid.UUID                   `json:"parentAgentId,omitempty,omitzero"`
	Position          int                         `json:"position,omitempty,omitzero"`
	SubAgents         []Agent                     `json:"subAgents,omitempty,omitzero"`
	UpdatedAt         time.Time                   `json:"updatedAt"`
}

type AgentCreate struct {
	validate[AgentCreate]
	ConfigJson        *ConfigJson                  `json:"configJson,omitempty,omitzero"`
	CredentialSource  enums.CredentialSource       `json:"credentialSource" validate:"required"`
	Description       string                       `json:"description" validate:"notblank"`
	GlobalInstruction *string                      `json:"globalInstruction,omitempty,omitzero"`
	Instruction       string                       `json:"instruction,omitempty,omitzero" validate:"notblank"`
	Mode              llmagent.Mode                `json:"mode" validate:"required"`
	ModelConfig       *genai.GenerateContentConfig `json:"modelConfig,omitempty,omitzero"`
	ModelCredentialID uuid.UUID                    `json:"modelCredentialId" validate:"required"`
	ModelName         string                       `json:"modelName"`
	Name              string                       `json:"name" validate:"notblank"`
	ParentAgentID     *uuid.UUID                   `json:"parentAgentId,omitempty,omitzero"`
}

type AgentDetail struct {
	ConfigJson        ConfigJson                  `json:"configJson,omitempty,omitzero"`
	CreatedAt         time.Time                   `json:"createdAt"`
	CredentialSource  enums.CredentialSource      `json:"credentialSource"`
	Description       string                      `json:"description"`
	GlobalInstruction string                      `json:"globalInstruction,omitempty,omitzero"`
	ID                uuid.UUID                   `json:"id"`
	Instruction       string                      `json:"instruction"`
	IsActive          bool                        `json:"isActive"`
	Mcps              []MCP                       `json:"mcps,omitempty,omitzero"`
	Mode              llmagent.Mode               `json:"mode"`
	ModelConfig       genai.GenerateContentConfig `json:"modelConfig,omitempty,omitzero"`
	ModelCredentialID uuid.UUID                   `json:"modelCredentialId"`
	ModelName         string                      `json:"modelName"`
	Name              string                      `json:"name"`
	ParentAgentID     uuid.UUID                   `json:"parentAgentId,omitempty,omitzero"`
	Position          int                         `json:"position,omitempty,omitzero"`
	Skills            []Skill                     `json:"skills,omitempty,omitzero"`
	SubAgents         []Agent                     `json:"subAgents,omitempty,omitzero"`
	UpdatedAt         time.Time                   `json:"updatedAt"`
}

type AgentUpdate struct {
	validate[AgentUpdate]
	ConfigJson        *ConfigJson                  `json:"configJson,omitempty,omitzero"`
	CredentialSource  *enums.CredentialSource      `json:"credentialSource,omitempty,omitzero"`
	Description       *string                      `json:"description,omitempty,omitzero"`
	GlobalInstruction *string                      `json:"globalInstruction,omitempty,omitzero"`
	Instruction       *string                      `json:"instruction,omitempty,omitzero"`
	IsActive          *bool                        `json:"isActive,omitempty,omitzero"`
	Mode              *llmagent.Mode               `json:"mode,omitempty,omitzero"`
	ModelConfig       *genai.GenerateContentConfig `json:"modelConfig,omitempty,omitzero"`
	ModelCredentialID *uuid.UUID                   `json:"modelCredentialId,omitempty,omitzero"`
	ModelName         *string                      `json:"modelName,omitempty,omitzero"`
}
