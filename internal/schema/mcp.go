package schema

import (
	"time"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/enums"
)

type AuthConfig map[string]any

type ConfirmationRules map[string]any

type MCP struct {
	AllowedTools        []string           `json:"allowedTools,omitempty,omitzero"`
	Args                []string           `json:"args,omitempty,omitzero"`
	AuthConfig          AuthConfig         `json:"authConfig,omitempty,omitzero"`
	AuthType            *enums.McpAuthType `json:"authType,omitempty,omitzero"`
	Command             *string            `json:"command,omitempty,omitzero"`
	ConfirmationRules   ConfirmationRules  `json:"confirmationRules,omitempty,omitzero"`
	CreatedAt           time.Time          `json:"createdAt"`
	Endpoint            *string            `json:"endpoint,omitempty,omitzero"`
	ID                  uuid.UUID          `json:"id"`
	IsActive            bool               `json:"isActive"`
	Name                string             `json:"name"`
	RequireConfirmation bool               `json:"requireConfirmation"`
	Transport           enums.McpTransport `json:"transport"`
	UpdatedAt           time.Time          `json:"updatedAt"`
	UserID              uuid.UUID          `json:"userId"`
}

type MCPCreate struct {
	validate[MCPCreate]
	AllowedTools        []string           `json:"allowedTools,omitempty,omitzero"`
	Args                []string           `json:"args,omitempty,omitzero"`
	AuthConfig          AuthConfig         `json:"authConfig,omitempty,omitzero" required_with:"AuthType"`
	AuthType            *enums.McpAuthType `json:"authType,omitempty,omitzero"`
	Command             *string            `json:"command,omitempty,omitzero" required_if:"Transport stdio"`
	ConfirmationRules   ConfirmationRules  `json:"confirmationRules,omitempty,omitzero"`
	Endpoint            string             `json:"endpoint,omitempty,omitzero" required_if:"Transport streamable_http"`
	IsActive            bool               `json:"isActive,omitempty,omitzero"`
	Name                string             `json:"name" validate:"notblank"`
	RequireConfirmation bool               `json:"requireConfirmation,omitempty,omitzero"`
	Transport           enums.McpTransport `json:"transport,omitempty,omitzero" validate:"required"`
	UserID              uuid.UUID          `json:"userId" validate:"required"`
}

type MCPUpdate struct {
	validate[MCPUpdate]
	AllowedTools        []string            `json:"allowedTools,omitempty,omitzero"`
	Args                []string            `json:"args,omitempty,omitzero"`
	AuthConfig          AuthConfig          `json:"authConfig,omitempty,omitzero" required_with:"AuthType"`
	AuthType            *enums.McpAuthType  `json:"authType,omitempty,omitzero"`
	Command             *string             `json:"command,omitempty,omitzero" required_if:"Transport stdio"`
	ConfirmationRules   ConfirmationRules   `json:"confirmationRules,omitempty,omitzero"`
	Endpoint            *string             `json:"endpoint,omitempty,omitzero" required_if:"Transport streamable_http"`
	IsActive            *bool               `json:"isActive,omitempty,omitzero"`
	Name                *string             `json:"name,omitempty,omitzero" validate:"notblank"`
	RequireConfirmation *bool               `json:"requireConfirmation,omitempty,omitzero"`
	Transport           *enums.McpTransport `json:"transport,omitempty,omitzero" validate:"required"`
}
