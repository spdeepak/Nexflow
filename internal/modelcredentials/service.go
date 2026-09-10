package modelcredentials

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/util"
	"github.com/spdeepak/nexflow/schema"
)

type (
	service struct {
		query Querier
	}

	Service interface {
		CreateAppModelCredential(ctx context.Context, arg schema.ModelCredentialCreate) (schema.ModelCredential, error)
		CreateUserModelCredential(ctx context.Context, userId uuid.UUID, arg schema.ModelCredentialCreate) (schema.ModelCredential, error)
		DeleteModelCredential(ctx context.Context, id uuid.UUID) error
		GetAvailableModelCredentials(ctx context.Context, userId uuid.UUID, active *bool) ([]schema.ModelCredential, error)
		GetModelCredential(ctx context.Context, id uuid.UUID) (GetModelCredentialRow, error)
		GetModelCredentialWithKey(ctx context.Context, id uuid.UUID) (ModelCredential, error)
		GetUserModelCredentials(ctx context.Context, arg GetUserModelCredentialsParams) ([]GetUserModelCredentialsRow, error)
		UpdateModelCredential(ctx context.Context, arg UpdateModelCredentialParams) (UpdateModelCredentialRow, error)
	}
)

func NewService(query Querier) Service {
	return &service{
		query: query,
	}
}

func (s *service) CreateAppModelCredential(ctx context.Context, arg schema.ModelCredentialCreate) (schema.ModelCredential, error) {
	id, _ := uuid.NewV7()
	createModel := CreateAppModelCredentialParams{
		ID:           id,
		Provider:     string(arg.Provider),
		ModelName:    arg.ModelName,
		Title:        arg.Title,
		BaseUrl:      sql.NullString{String: arg.BaseUrl, Valid: true},
		ApiKeyCipher: util.GetSQLNullString(&arg.ApiKey),
		KeyVersion:   1,
	}
	slog.InfoContext(ctx, "CreateAppModelCredential", "model", createModel)
	credential, err := s.query.CreateAppModelCredential(ctx, createModel)
	if err != nil {
		if strings.Contains(err.Error(), "model_credentials_owner_check") {
			return schema.ModelCredential{}, fmt.Errorf("app level model credential cannot not be linked to a user")
		}
		return schema.ModelCredential{}, err
	}
	return schema.ModelCredential{
		BaseUrl:    credential.BaseUrl.String,
		CreatedAt:  credential.CreatedAt,
		Title:      credential.Title,
		ID:         credential.ID,
		IsActive:   credential.IsActive,
		KeyVersion: int(credential.KeyVersion),
		ModelName:  credential.ModelName,
		Provider:   credential.Provider,
		UpdatedAt:  credential.UpdatedAt,
	}, nil
}

func (s *service) CreateUserModelCredential(ctx context.Context, userId uuid.UUID, arg schema.ModelCredentialCreate) (schema.ModelCredential, error) {
	id, _ := uuid.NewV7()
	createModel := CreateUserModelCredentialParams{
		ID:           id,
		UserID:       userId,
		Provider:     string(arg.Provider),
		ModelName:    arg.ModelName,
		Title:        arg.Title,
		BaseUrl:      sql.NullString{String: arg.BaseUrl, Valid: true},
		ApiKeyCipher: util.GetSQLNullString(&arg.ApiKey),
		KeyVersion:   1,
	}
	slog.InfoContext(ctx, "CreateUserModelCredential", "model", createModel)
	credential, err := s.query.CreateUserModelCredential(ctx, createModel)
	if err != nil {
		return schema.ModelCredential{}, err
	}
	return schema.ModelCredential{
		BaseUrl:    credential.BaseUrl.String,
		CreatedAt:  credential.CreatedAt,
		Title:      credential.Title,
		ID:         credential.ID,
		IsActive:   credential.IsActive,
		KeyVersion: int(credential.KeyVersion),
		ModelName:  credential.ModelName,
		Provider:   credential.Provider,
		UpdatedAt:  credential.UpdatedAt,
	}, nil
}

func (s *service) DeleteModelCredential(ctx context.Context, id uuid.UUID) error {
	err := s.query.DeleteModelCredential(ctx, id)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to delete model credential", "error", err)
		return err
	}
	return nil
}

func (s *service) GetAvailableModelCredentials(ctx context.Context, userId uuid.UUID, active *bool) ([]schema.ModelCredential, error) {
	param := GetAvailableModelCredentialsParams{
		UserID: userId,
	}
	if active == nil {
		param.IsActive = sql.NullBool{
			Valid: false,
		}
	} else {
		param.IsActive = sql.NullBool{
			Bool:  *active,
			Valid: true,
		}
	}
	credentials, err := s.query.GetAvailableModelCredentials(ctx, param)
	if err != nil {
		return nil, err
	}
	models := make([]schema.ModelCredential, len(credentials))
	for index, credential := range credentials {
		models[index] = schema.ModelCredential{
			BaseUrl:    credential.BaseUrl.String,
			CreatedAt:  credential.CreatedAt,
			Title:      credential.Title,
			ID:         credential.ID,
			IsActive:   credential.IsActive,
			KeyVersion: int(credential.KeyVersion),
			ModelName:  credential.ModelName,
			Provider:   credential.Provider,
			Scope:      credential.Scope,
			UpdatedAt:  credential.UpdatedAt,
			UserID:     credential.UserID,
		}
	}
	return models, nil
}

func (s *service) GetModelCredential(ctx context.Context, id uuid.UUID) (GetModelCredentialRow, error) {
	row, err := s.query.GetModelCredential(ctx, id)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get model credential", "id", id, "error", err)
		return GetModelCredentialRow{}, err
	}
	return row, nil
}

func (s *service) GetModelCredentialWithKey(ctx context.Context, id uuid.UUID) (ModelCredential, error) {
	row, err := s.query.GetModelCredentialWithKey(ctx, id)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get model credential with key", "id", id, "error", err)
		return ModelCredential{}, err
	}
	return ModelCredential{
		ID:           row.ID,
		Title:        row.Title,
		UserID:       row.UserID,
		Provider:     row.Provider,
		ModelName:    row.ModelName,
		BaseUrl:      row.BaseUrl,
		Scope:        row.Scope,
		ApiKeyCipher: row.ApiKeyCipher,
		KeyVersion:   row.KeyVersion,
		IsActive:     row.IsActive,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}

func (s *service) GetUserModelCredentials(ctx context.Context, arg GetUserModelCredentialsParams) ([]GetUserModelCredentialsRow, error) {
	rows, err := s.query.GetUserModelCredentials(ctx, arg)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get user model credentials", "error", err)
		return nil, err
	}
	return rows, nil
}

func (s *service) UpdateModelCredential(ctx context.Context, arg UpdateModelCredentialParams) (UpdateModelCredentialRow, error) {
	row, err := s.query.UpdateModelCredential(ctx, arg)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to update model credential", "error", err)
		return UpdateModelCredentialRow{}, err
	}
	return row, nil
}
