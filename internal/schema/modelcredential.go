package schema

import (
	"time"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/enums"
)

type ModelCredentialProvider string

const (
	ModelCredentialCreateProviderAnthropic ModelCredentialProvider = "anthropic"
	ModelCredentialCreateProviderGoogle    ModelCredentialProvider = "google"
	ModelCredentialCreateProviderOllama    ModelCredentialProvider = "ollama"
	ModelCredentialCreateProviderOpenai    ModelCredentialProvider = "openai"
)

type ExtraConfig map[string]any

type ModelCredential struct {
	BaseUrl     string                `json:"baseUrl"`
	CreatedAt   time.Time             `json:"createdAt"`
	ExtraConfig *ExtraConfig          `json:"extraConfig,omitempty,omitzero"`
	ID          uuid.UUID             `json:"id"`
	IsActive    bool                  `json:"isActive"`
	KeyVersion  int                   `json:"keyVersion"`
	ModelName   string                `json:"modelName"`
	Provider    string                `json:"provider"`
	Scope       enums.CredentialScope `json:"scope"`
	Title       string                `json:"title"`
	UpdatedAt   time.Time             `json:"updatedAt"`
	UserID      uuid.UUID             `json:"userId"`
}

type ModelCredentialCreate struct {
	validate[ModelCredentialCreate]
	ApiKey      string                  `json:"apiKey"`
	BaseUrl     string                  `json:"baseUrl" validate:"required,url"`
	ExtraConfig ExtraConfig             `json:"extraConfig,omitempty,omitzero"`
	ModelName   string                  `json:"modelName" validate:"required,min=1"`
	Provider    ModelCredentialProvider `json:"provider" validate:"required"`
	Scope       enums.CredentialScope   `json:"scope" validate:"required"`
	Title       string                  `json:"title" validate:"required,min=1"`
}

type ModelCredentialUpdate struct {
	validate[ModelCredentialUpdate]
	ApiKey      *string                  `json:"apiKey,omitempty,omitzero" validate:"required"`
	BaseUrl     *string                  `json:"baseUrl,omitempty,omitzero" validate:"required,url"`
	ExtraConfig *ExtraConfig             `json:"extraConfig,omitempty,omitzero"`
	IsActive    bool                     `json:"isActive"`
	ModelName   *string                  `json:"modelName,omitempty,omitzero" validate:"required,min=1"`
	Provider    *ModelCredentialProvider `json:"provider,omitempty,omitzero" validate:"required,min=1"`
	Title       *string                  `json:"title,omitempty,omitzero" validate:"required,min=1"`
}

type ModelOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}
