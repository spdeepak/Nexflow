package application

import (
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/modelcredentials"
	"github.com/spdeepak/nexflow/internal/schema"
	"github.com/spdeepak/nexflow/internal/util"
)

func (a *App) CreateModelCredential(params schema.ModelCredentialCreate) error {
	if err := params.Validate(); err != nil {
		return err
	}
	err := a.Validate(params)
	if err != nil {
		return err
	}

	if params.Scope == enums.CredentialScopeApp {
		_, err = a.modelService.CreateAppModelCredential(a.ctx, params)
	} else {
		_, err = a.modelService.CreateUserModelCredential(a.ctx, a.deviceID, params)
	}
	return err
}

func (a *App) GetModelCredentials() ([]schema.ModelCredential, error) {
	modelCredentials, err := a.modelService.GetAvailableModelCredentials(a.ctx, a.deviceID, nil)
	if err != nil {
		slog.Error("Error getting model credentials", "error", err)
		return nil, err
	}
	return modelCredentials, nil
}

func (a *App) GetModelOptions() []schema.ModelOption {
	creds, err := a.modelService.GetAvailableModelCredentials(a.ctx, a.deviceID, nil)
	if err != nil {
		slog.Error("Error getting model options", "error", err)
		return nil
	}
	modelOptions := make([]schema.ModelOption, len(creds))
	for index, model := range creds {
		modelOptions[index] = schema.ModelOption{
			Value: model.ID.String(),
			Label: fmt.Sprintf("%s (%s/%s)", model.Title, model.Provider, model.ModelName),
		}
	}
	return modelOptions
}

func (a *App) DeleteModel(modelId string) error {
	id, err := uuid.Parse(modelId)
	if err != nil {
		return fmt.Errorf("invalid model id: %s", modelId)
	}
	err = a.modelService.DeleteModelCredential(a.ctx, id)
	if err != nil {
		return err
	}
	return nil
}

func (a *App) UpdateModelCredential(modelId string, params schema.ModelCredentialUpdate) error {
	if err := params.Validate(); err != nil {
		return err
	}
	id, err := uuid.Parse(modelId)
	if err != nil {
		return fmt.Errorf("invalid model id: %s", modelId)
	}
	arg := modelcredentials.UpdateModelCredentialParams{
		ID:           id,
		Provider:     util.GetSQLNullString((*string)(params.Provider)),
		ModelName:    util.GetSQLNullString(params.ModelName),
		Title:        util.GetSQLNullString(params.Title),
		BaseUrl:      util.GetSQLNullString(params.BaseUrl),
		ApiKeyCipher: util.GetSQLNullString(params.ApiKey),
		IsActive:     params.IsActive,
	}
	_, err = a.modelService.UpdateModelCredential(a.ctx, arg)
	return err
}
